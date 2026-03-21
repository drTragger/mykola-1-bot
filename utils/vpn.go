package utils

import (
	"fmt"
	"os/user"
	"strconv"
	"strings"
	"time"
)

func humanizeDurationShort(d time.Duration) string {
	if d < 0 {
		return "н/д"
	}

	seconds := int(d.Seconds())
	if seconds < 60 {
		return fmt.Sprintf("%dс", seconds)
	}

	minutes := seconds / 60
	if minutes < 60 {
		return fmt.Sprintf("%dхв", minutes)
	}

	hours := minutes / 60
	minutes = minutes % 60
	if hours < 24 {
		return fmt.Sprintf("%dг %dхв", hours, minutes)
	}

	days := hours / 24
	hours = hours % 24
	return fmt.Sprintf("%dд %dг", days, hours)
}

func getWireGuardStatus() string {
	return getSystemdServiceStatus("wg-quick@wg0")
}

func getWGInterfaceIP() string {
	out, err := runCommand(2, "ip", "-4", "-o", "addr", "show", "dev", "wg0")
	if err != nil || out == "" {
		return "н/д"
	}

	fields := strings.Fields(out)
	if len(fields) < 4 {
		return "н/д"
	}

	return fields[3]
}

func getWGDumpData() (endpoint, handshakeAgo, rx, tx string) {
	out, err := runSudoCommand(2, "wg", "show", "wg0", "dump")
	if err != nil || out == "" {
		return "н/д", "н/д", "н/д", "н/д"
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		return "н/д", "н/д", "н/д", "н/д"
	}

	fields := strings.Fields(lines[1])
	if len(fields) < 8 {
		return "н/д", "н/д", "н/д", "н/д"
	}

	endpoint = fields[2]

	handshakeUnix, err := strconv.ParseInt(fields[4], 10, 64)
	if err != nil || handshakeUnix == 0 {
		handshakeAgo = "ніколи"
	} else {
		handshakeAgo = humanizeDurationShort(time.Since(time.Unix(handshakeUnix, 0))) + " тому"
	}

	rxBytes, err := strconv.ParseUint(fields[5], 10, 64)
	if err != nil {
		rx = "н/д"
	} else {
		rx = formatBytesIEC(rxBytes)
	}

	txBytes, err := strconv.ParseUint(fields[6], 10, 64)
	if err != nil {
		tx = "н/д"
	} else {
		tx = formatBytesIEC(txBytes)
	}

	return endpoint, handshakeAgo, rx, tx
}

func getQBittorrentUserInfo() (username string, uid string) {
	u, err := user.Lookup("qbittorrent")
	if err != nil {
		return "qbittorrent", ""
	}

	return u.Username, u.Uid
}

func hasVPNRuleForQBittorrent() bool {
	_, uid := getQBittorrentUserInfo()
	if uid == "" {
		return false
	}

	out, err := runCommand(2, "ip", "rule")
	if err != nil {
		return false
	}

	expected := fmt.Sprintf("uidrange %s-%s lookup vpn", uid, uid)
	return strings.Contains(out, expected)
}

func getVPNRouteTable() string {
	out, err := runCommand(2, "ip", "route", "show", "table", "vpn")
	if err != nil || strings.TrimSpace(out) == "" {
		return "❌ відсутній"
	}

	return strings.TrimSpace(out)
}

func getQBittorrentServiceUser() string {
	serviceNames := []string{
		"qbittorrent",
		"qbittorrent.service",
		"qbittorrent-nox",
		"qbittorrent-nox.service",
	}

	for _, service := range serviceNames {
		out, err := runCommand(2, "systemctl", "show", service, "--property=User", "--value")
		if err == nil && strings.TrimSpace(out) != "" {
			return strings.TrimSpace(out)
		}
	}

	return "н/д"
}

func getQBittorrentServiceStatus() string {
	return getFirstActiveServiceStatus(
		"qbittorrent",
		"qbittorrent.service",
		"qbittorrent-nox",
		"qbittorrent-nox.service",
	)
}

func getQBittorrentConfigPaths() []string {
	return []string{
		"/home/qbittorrent/.config/qBittorrent/qBittorrent.conf",
		"/home/qbittorrent/.local/share/qBittorrent/qBittorrent.conf",
		"/home/qbittorrent/.config/qBittorrent/config/qBittorrent.conf",
	}
}

func getQBittorrentInterfaceBinding() string {
	data := readQBittorrentConfig()
	if data == "" {
		return "н/д"
	}

	lines := strings.Split(data, "\n")
	keys := []string{
		"Connection\\Interface=",
		"Connection\\InterfaceName=",
		"Session\\Interface=",
		"Session\\InterfaceName=",
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		for _, key := range keys {
			if strings.HasPrefix(line, key) {
				return strings.TrimSpace(strings.TrimPrefix(line, key))
			}
		}
	}

	return "н/д"
}

func getQBittorrentWebUIAddress() string {
	data := readQBittorrentConfig()
	if data == "" {
		return "н/д"
	}

	lines := strings.Split(data, "\n")
	address := "0.0.0.0"
	port := "8080"

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "WebUI\\Address=") {
			address = strings.TrimSpace(strings.TrimPrefix(line, "WebUI\\Address="))
		}

		if strings.HasPrefix(line, "WebUI\\Port=") {
			port = strings.TrimSpace(strings.TrimPrefix(line, "WebUI\\Port="))
		}
	}

	return fmt.Sprintf("%s:%s", address, port)
}

func runSudoCommand(timeoutSec int, cmd string, args ...string) (string, error) {
	allArgs := append([]string{"-n", cmd}, args...)
	return runCommand(timeoutSec, "sudo", allArgs...)
}

func readQBittorrentConfig() string {
	paths := []string{
		"/home/qbittorrent/.config/qBittorrent/qBittorrent.conf",
		"/home/qbittorrent/.local/share/qBittorrent/qBittorrent.conf",
		"/home/qbittorrent/.config/qBittorrent/config/qBittorrent.conf",
		"/home/qbittorrent/.config/qBittorrent/qBittorrent-data.conf",
	}

	for _, path := range paths {
		out, err := runSudoCommand(2, "cat", path)
		if err == nil && strings.TrimSpace(out) != "" {
			return out
		}
	}

	return ""
}

func GetVPNSummaryShort() string {
	status := getWireGuardStatus()
	if status != "✅" {
		return "🔐 *VPN:* ❌"
	}

	_, handshakeAgo, _, _ := getWGDumpData()

	routeStatus := "❌"
	if strings.Contains(getVPNRouteTable(), "default dev wg0") {
		routeStatus = "OK"
	}

	return fmt.Sprintf("🔐 *VPN:* ✅ handshake %s, qBittorrent route: %s", handshakeAgo, routeStatus)
}

func GetVPNDetails() string {
	status := getWireGuardStatus()
	wgIP := getWGInterfaceIP()
	endpoint, handshakeAgo, rx, tx := getWGDumpData()
	routeTable := getVPNRouteTable()

	qbitServiceStatus := getQBittorrentServiceStatus()
	qbitServiceUser := getQBittorrentServiceUser()
	qbitBinding := getQBittorrentInterfaceBinding()
	qbitWebUI := getQBittorrentWebUIAddress()

	ruleStatus := "❌"
	if hasVPNRuleForQBittorrent() {
		ruleStatus = "✅"
	}

	routeStatus := "❌"
	if strings.Contains(routeTable, "default dev wg0") {
		routeStatus = "✅"
	}

	bindStatus := "❌"
	if qbitBinding == "wg0" {
		bindStatus = "✅"
	}

	return fmt.Sprintf(`🔐 *VPN / WireGuard*

• *wg-quick@wg0:* %s
• *Інтерфейс:* wg0
• *IP wg0:* %s
• *Endpoint:* %s
• *Handshake:* %s
• *Received:* %s
• *Sent:* %s

📦 *qBittorrent*
• *Сервіс:* %s
• *User:* %s
• *Binding:* %s (%s)
• *Web UI:* %s

🛣️ *Routing*
• *ip rule для qbittorrent:* %s
• *table vpn:* %s
• *route через wg0:* %s`,
		status,
		wgIP,
		endpoint,
		handshakeAgo,
		rx,
		tx,
		qbitServiceStatus,
		qbitServiceUser,
		qbitBinding,
		bindStatus,
		qbitWebUI,
		ruleStatus,
		routeTable,
		routeStatus,
	)
}
