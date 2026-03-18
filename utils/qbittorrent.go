package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"mykola-1-bot/config"
)

type Torrent struct {
	Name     string  `json:"name"`
	Progress float64 `json:"progress"`
	State    string  `json:"state"`
	Dlspeed  int64   `json:"dlspeed"`
	Upspeed  int64   `json:"upspeed"`
	Size     int64   `json:"size"`
}

var qbClient = &http.Client{
	Timeout: 5 * time.Second,
}

func GetTorrentsStatus() string {
	torrents, err := GetTorrents()
	if err != nil {
		return "❌ Не вдалося отримати список торентів"
	}

	if len(torrents) == 0 {
		return "📭 Немає активних торентів"
	}

	var b strings.Builder
	b.WriteString("🎬 *qBittorrent*\n\n")

	for i, t := range torrents {
		if i >= 10 {
			b.WriteString("\n...і ще інші")
			break
		}

		progress := int(t.Progress * 100)

		b.WriteString(fmt.Sprintf(
			`• *%s*
  %s | %d%%
  ↓ %s | ↑ %s

`,
			truncate(t.Name, 40),
			mapState(t.State),
			progress,
			formatSpeed(t.Dlspeed),
			formatSpeed(t.Upspeed),
		))
	}

	return b.String()
}

func qbLogin() (*http.Client, error) {
	data := url.Values{}
	data.Set("username", config.Cfg.QBittorrent.Username)
	data.Set("password", config.Cfg.QBittorrent.Password)

	req, _ := http.NewRequest("POST", config.Cfg.QBittorrent.URL+"/api/v2/auth/login", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := qbClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return qbClient, nil
}

func GetTorrents() ([]Torrent, error) {
	client, err := qbLogin()
	if err != nil {
		return nil, err
	}

	req, _ := http.NewRequest("GET", config.Cfg.QBittorrent.URL+"/api/v2/torrents/info", nil)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var torrents []Torrent
	if err := json.Unmarshal(body, &torrents); err != nil {
		return nil, err
	}

	return torrents, nil
}

func formatSpeed(b int64) string {
	if b <= 0 {
		return "0"
	}
	kb := float64(b) / 1024
	mb := kb / 1024

	if mb >= 1 {
		return fmt.Sprintf("%.1f MB/s", mb)
	}
	return fmt.Sprintf("%.0f KB/s", kb)
}

func mapState(state string) string {
	switch state {
	case "downloading":
		return "⬇️ Завантаження"
	case "uploading":
		return "⬆️ Сід"
	case "pausedDL", "pausedUP":
		return "⏸ Пауза"
	case "stalledDL":
		return "⏳ Очікує"
	case "error":
		return "❌ Помилка"
	default:
		return state
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
