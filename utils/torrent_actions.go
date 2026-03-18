package utils

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"mykola-1-bot/config"
)

func PauseTorrent(hash string) error {
	return torrentAction("pause", hash)
}

func ResumeTorrent(hash string) error {
	return torrentAction("resume", hash)
}

func FindTorrentByIndex(index int) (*Torrent, error) {
	torrents, err := GetSortedTorrents()
	if err != nil {
		return nil, err
	}

	if index <= 0 || index > len(torrents) {
		return nil, fmt.Errorf("невірний номер")
	}

	return &torrents[index-1], nil
}

func GetSortedTorrents() ([]Torrent, error) {
	qbCacheMu.Lock()
	if time.Since(qbCacheAt) < qbCacheTTL && qbCacheTorrents != nil {
		cached := make([]Torrent, len(qbCacheTorrents))
		copy(cached, qbCacheTorrents)
		qbCacheMu.Unlock()

		sortTorrents(cached)
		return cached, nil
	}
	qbCacheMu.Unlock()

	torrents, err := getTorrentsWithRelogin()
	if err != nil {
		return nil, err
	}

	sortTorrents(torrents)

	qbCacheMu.Lock()
	qbCacheTorrents = make([]Torrent, len(torrents))
	copy(qbCacheTorrents, torrents)
	qbCacheAt = time.Now()
	qbCacheMu.Unlock()

	return torrents, nil
}

func IsPausedState(state string) bool {
	switch state {
	case "pausedDL", "pausedUP":
		return true
	default:
		return false
	}
}

func FindTorrentByHash(hash string) (*Torrent, error) {
	torrents, err := GetTorrents()
	if err != nil {
		return nil, err
	}

	for _, t := range torrents {
		if t.Hash == hash {
			return &t, nil
		}
	}

	return nil, fmt.Errorf("торент не знайдено")
}

func torrentAction(action string, hash string) error {
	data := url.Values{}
	data.Set("hashes", hash)

	req, err := http.NewRequest(
		"POST",
		config.Cfg.QBittorrent.URL+"/api/v2/torrents/"+action,
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := qbClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qb %s failed: %s", action, string(body))
	}

	InvalidateTorrentsCache()

	return nil
}
