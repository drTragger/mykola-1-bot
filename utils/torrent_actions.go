package utils

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"mykola-1-bot/config"
)

func PauseTorrent(hash string) error {
	return torrentAction("pause", hash)
}

func ResumeTorrent(hash string) error {
	return torrentAction("resume", hash)
}

func FindTorrentByIndex(index int) (*Torrent, error) {
	torrents, err := GetTorrents()
	if err != nil {
		return nil, err
	}

	if index <= 0 || index > len(torrents) {
		return nil, fmt.Errorf("невірний номер")
	}

	return &torrents[index-1], nil
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
