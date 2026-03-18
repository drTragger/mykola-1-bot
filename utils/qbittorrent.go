package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"mykola-1-bot/config"
)

var (
	qbClient *http.Client

	qbCacheMu       sync.Mutex
	qbCacheTorrents []Torrent
	qbCacheAt       time.Time
	qbCacheTTL      = 5 * time.Second
)

func init() {
	jar, _ := cookiejar.New(nil)

	qbClient = &http.Client{
		Jar:     jar,
		Timeout: 5 * time.Second,
	}
}

func GetTorrentsStatus() string {
	torrents, err := GetSortedTorrents()
	if err != nil {
		return fmt.Sprintf("❌ Помилка qBittorrent: `%v`", err)
	}

	if len(torrents) == 0 {
		return "📭 *qBittorrent*\n\nНемає торрентів."
	}

	total := len(torrents)
	activeDownloading := 0
	activeUploading := 0

	var totalDl int64
	var totalUl int64

	for _, t := range torrents {
		totalDl += t.Dlspeed
		totalUl += t.Upspeed

		if IsDownloadingState(t.State) {
			activeDownloading++
		}
		if isUploadingState(t.State) {
			activeUploading++
		}
	}

	var b strings.Builder
	b.WriteString("🎬 *qBittorrent*\n\n")
	b.WriteString(fmt.Sprintf(
		"📦 *Всього:* %d\n⬇️ *Завантажуються:* %d\n⬆️ *Роздаються:* %d\n🚀 *Швидкість:* ↓ %s | ↑ %s\n\n",
		total,
		activeDownloading,
		activeUploading,
		formatSpeedOrDash(totalDl),
		formatSpeedOrDash(totalUl),
	))

	limit := 10
	for i, t := range torrents {
		index := i + 1
		if i >= limit {
			b.WriteString(fmt.Sprintf("\n_...і ще %d торрент(ів)_", len(torrents)-limit))
			break
		}

		name := fmt.Sprintf("%d. %s", index, EscapeMarkdown(truncate(t.Name, 45)))
		progress := int(t.Progress * 100)
		size := formatBytesIECInt64(effectiveSize(t.Size, t.TotalSize))
		state := mapState(t.State)
		bar := progressBar(progress, 10)

		line := fmt.Sprintf(
			"*%s*\n%s %d%% • %s\n⬇️ %s • ⬆️ %s",
			name,
			bar,
			progress,
			state,
			formatSpeedOrDash(t.Dlspeed),
			formatSpeedOrDash(t.Upspeed),
		)

		if size != "0 B" {
			line += fmt.Sprintf(" • %s", size)
		}

		if IsDownloadingState(t.State) && t.Eta > 0 {
			line += fmt.Sprintf("\n⏳ ETA: %s", formatETA(t.Eta))
		}

		if t.NumSeeds > 0 || t.NumLeechs > 0 {
			line += fmt.Sprintf("\n🌱 %d сидів • 🧲 %d качають", t.NumSeeds, t.NumLeechs)
		}

		if t.NumSeeds == 0 && IsDownloadingState(t.State) {
			line += "\n⚠️ Немає сидів — може не скачатися"
		}

		b.WriteString(line + "\n\n")
	}

	return b.String()
}

func GetTorrents() ([]Torrent, error) {
	qbCacheMu.Lock()
	if time.Since(qbCacheAt) < qbCacheTTL && qbCacheTorrents != nil {
		cached := make([]Torrent, len(qbCacheTorrents))
		copy(cached, qbCacheTorrents)
		qbCacheMu.Unlock()
		return cached, nil
	}
	qbCacheMu.Unlock()

	torrents, err := getTorrentsWithRelogin()
	if err != nil {
		return nil, err
	}

	qbCacheMu.Lock()
	qbCacheTorrents = make([]Torrent, len(torrents))
	copy(qbCacheTorrents, torrents)
	qbCacheAt = time.Now()
	qbCacheMu.Unlock()

	return torrents, nil
}

func InvalidateTorrentsCache() {
	qbCacheMu.Lock()
	defer qbCacheMu.Unlock()

	qbCacheTorrents = nil
	qbCacheAt = time.Time{}
}

func getTorrentsWithRelogin() ([]Torrent, error) {
	torrents, status, err := fetchTorrents()
	if err == nil {
		return torrents, nil
	}

	if status == http.StatusForbidden || status == http.StatusUnauthorized {
		if err := qbLogin(); err != nil {
			return nil, err
		}
		return fetchTorrentsAfterLogin()
	}

	return nil, err
}

func fetchTorrentsAfterLogin() ([]Torrent, error) {
	torrents, _, err := fetchTorrents()
	return torrents, err
}

func qbLogin() error {
	data := url.Values{}
	data.Set("username", config.Cfg.QBittorrent.Username)
	data.Set("password", config.Cfg.QBittorrent.Password)

	req, err := http.NewRequest("POST", config.Cfg.QBittorrent.URL+"/api/v2/auth/login", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := qbClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "Ok.") {
		return fmt.Errorf("qb login failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}

func fetchTorrents() ([]Torrent, int, error) {
	req, err := http.NewRequest("GET", config.Cfg.QBittorrent.URL+"/api/v2/torrents/info", nil)
	if err != nil {
		return nil, 0, err
	}

	resp, err := qbClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, fmt.Errorf("qb torrents request failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var torrents []Torrent
	if err := json.Unmarshal(body, &torrents); err != nil {
		return nil, resp.StatusCode, err
	}

	return torrents, resp.StatusCode, nil
}

func sortTorrents(torrents []Torrent) {
	sort.SliceStable(torrents, func(i, j int) bool {
		pi := torrentPriority(torrents[i])
		pj := torrentPriority(torrents[j])

		if pi != pj {
			return pi < pj
		}

		if torrents[i].Dlspeed != torrents[j].Dlspeed {
			return torrents[i].Dlspeed > torrents[j].Dlspeed
		}

		if torrents[i].Upspeed != torrents[j].Upspeed {
			return torrents[i].Upspeed > torrents[j].Upspeed
		}

		return torrents[i].Name < torrents[j].Name
	})
}

func torrentPriority(t Torrent) int {
	switch {
	case IsDownloadingState(t.State):
		return 0
	case isUploadingState(t.State):
		return 1
	case strings.HasPrefix(t.State, "paused"):
		return 3
	default:
		return 2
	}
}

func IsDownloadingState(state string) bool {
	switch state {
	case "downloading", "metaDL", "forcedDL", "stalledDL", "checkingDL", "queuedDL":
		return true
	default:
		return false
	}
}

func isUploadingState(state string) bool {
	switch state {
	case "uploading", "forcedUP", "stalledUP", "queuedUP":
		return true
	default:
		return false
	}
}

func mapState(state string) string {
	switch state {
	case "downloading":
		return "⬇️ Завантаження"
	case "metaDL":
		return "🧲 Отримання метаданих"
	case "forcedDL":
		return "⬇️ Форсоване завантаження"
	case "stalledDL":
		return "⏳ Очікує пірів"
	case "checkingDL", "checkingUP":
		return "🔍 Перевірка"
	case "queuedDL":
		return "📥 У черзі"
	case "uploading":
		return "⬆️ Роздача"
	case "forcedUP":
		return "⬆️ Форсована роздача"
	case "stalledUP":
		return "🌱 Сід без активності"
	case "queuedUP":
		return "📤 У черзі"
	case "pausedDL", "pausedUP":
		return "⏸ Пауза"
	case "moving":
		return "📂 Переміщення"
	case "error", "missingFiles":
		return "❌ Помилка"
	default:
		return state
	}
}

func formatSpeed(b int64) string {
	if b <= 0 {
		return "0"
	}

	kb := float64(b) / 1024
	mb := kb / 1024
	gb := mb / 1024

	switch {
	case gb >= 1:
		return fmt.Sprintf("%.2f GB/s", gb)
	case mb >= 1:
		return fmt.Sprintf("%.1f MB/s", mb)
	default:
		return fmt.Sprintf("%.0f KB/s", kb)
	}
}

func formatSpeedOrDash(b int64) string {
	if b <= 0 {
		return "—"
	}
	return formatSpeed(b)
}

func formatETA(seconds int64) string {
	if seconds <= 0 || seconds > 8640000 {
		return "—"
	}

	d := seconds / 86400
	h := (seconds % 86400) / 3600
	m := (seconds % 3600) / 60

	switch {
	case d > 0:
		return fmt.Sprintf("%d д %d год", d, h)
	case h > 0:
		return fmt.Sprintf("%d год %d хв", h, m)
	default:
		return fmt.Sprintf("%d хв", m)
	}
}

func progressBar(percent int, width int) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	filled := percent * width / 100
	if filled > width {
		filled = width
	}

	return "[" + strings.Repeat("■", filled) + strings.Repeat("□", width-filled) + "]"
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "..."
}

func EscapeMarkdown(s string) string {
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"`", "\\`",
	)
	return replacer.Replace(s)
}

func effectiveSize(size int64, totalSize int64) int64 {
	if totalSize > 0 {
		return totalSize
	}
	return size
}

func formatBytesIECInt64(b int64) string {
	if b <= 0 {
		return "0 B"
	}

	const (
		KiB = 1024
		MiB = 1024 * KiB
		GiB = 1024 * MiB
		TiB = 1024 * GiB
	)

	switch {
	case b >= TiB:
		return fmt.Sprintf("%.2f TiB", float64(b)/TiB)
	case b >= GiB:
		return fmt.Sprintf("%.2f GiB", float64(b)/GiB)
	case b >= MiB:
		return fmt.Sprintf("%.2f MiB", float64(b)/MiB)
	case b >= KiB:
		return fmt.Sprintf("%.2f KiB", float64(b)/KiB)
	default:
		return fmt.Sprintf("%d B", b)
	}
}
