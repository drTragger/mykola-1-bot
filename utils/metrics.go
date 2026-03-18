package utils

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	gonet "github.com/shirou/gopsutil/v3/net"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

var (
	netSpeedMu      sync.Mutex
	prevNetRxBytes  uint64
	prevNetTxBytes  uint64
	prevNetSampleAt time.Time
)

func formatUptime(seconds uint64) string {
	d := time.Duration(seconds) * time.Second
	return fmt.Sprintf("%dд %02dг %02dх", d/(24*time.Hour), (d%(24*time.Hour))/time.Hour, (d%time.Hour)/time.Minute)
}

func formatBytesIEC(b uint64) string {
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

func formatBytesPerSecondIEC(bps float64) string {
	const (
		KiB = 1024.0
		MiB = 1024.0 * KiB
		GiB = 1024.0 * MiB
	)

	switch {
	case bps >= GiB:
		return fmt.Sprintf("%.2f GiB/s", bps/GiB)
	case bps >= MiB:
		return fmt.Sprintf("%.2f MiB/s", bps/MiB)
	case bps >= KiB:
		return fmt.Sprintf("%.2f KiB/s", bps/KiB)
	default:
		return fmt.Sprintf("%.0f B/s", bps)
	}
}

func readExternalURL(url string) string {
	client := &http.Client{Timeout: 3 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return "н/д"
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "н/д"
	}
	return strings.TrimSpace(string(data))
}

func runCommandAndExtract(pattern string, cmd string, args ...string) string {
	out, err := exec.Command(cmd, args...).Output()
	if err != nil {
		return "н/д"
	}
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(string(out))
	if len(match) >= 2 {
		return match[1]
	}
	return "н/д"
}

func runCommand(timeoutSec int, cmd string, args ...string) (string, error) {
	commandArgs := append([]string{strconv.Itoa(timeoutSec), cmd}, args...)
	ctxCmd := exec.Command("timeout", commandArgs...)
	out, err := ctxCmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func getCPUUsage() (string, int) {
	percents, err := cpu.Percent(0, false)
	if err != nil || len(percents) == 0 {
		return "н/д", 0
	}
	count, _ := cpu.Counts(true)
	return fmt.Sprintf("%.2f%%", percents[0]), count
}

func getCPUFreq() string {
	paths := []string{
		"/sys/devices/system/cpu/cpu0/cpufreq/scaling_cur_freq",
		"/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_cur_freq",
	}

	for _, path := range paths {
		line, err := readFirstLine(path)
		if err != nil || line == "" {
			continue
		}

		v, err := strconv.ParseFloat(line, 64)
		if err != nil {
			continue
		}

		mhz := v / 1000.0
		return fmt.Sprintf("%.0f MHz", mhz)
	}

	out, err := runCommand(2, "vcgencmd", "measure_clock", "arm")
	if err == nil {
		re := regexp.MustCompile(`frequency$begin:math:text$48$end:math:text$=([0-9]+)`)
		m := re.FindStringSubmatch(out)
		if len(m) >= 2 {
			v, err := strconv.ParseFloat(m[1], 64)
			if err == nil {
				return fmt.Sprintf("%.0f MHz", v/1_000_000.0)
			}
		}
	}

	return "н/д"
}

func getMemoryUsage() string {
	vm, err := mem.VirtualMemory()
	if err != nil {
		return "н/д"
	}
	return fmt.Sprintf("%.2f%% (%.1f GB / %.1f GB)",
		vm.UsedPercent, float64(vm.Used)/1e9, float64(vm.Total)/1e9)
}

func humanSwap(sizeKiB, usedKiB uint64) string {
	if sizeKiB == 0 {
		return "н/д"
	}
	pct := float64(usedKiB) / float64(sizeKiB) * 100
	gb := func(kib uint64) float64 { return float64(kib) * 1024 / 1e9 }
	return fmt.Sprintf("%.2f%% (%.1f GB / %.1f GB)", pct, gb(usedKiB), gb(sizeKiB))
}

func getSwapDetailed() (string, string) {
	f, err := os.Open("/proc/swaps")
	if err != nil {
		return "н/д", "н/д"
	}
	defer f.Close()

	zram := "н/д"
	swapfile := "н/д"

	sc := bufio.NewScanner(f)
	first := true
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if first {
			first = false
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		filename := fields[0]
		sizeKiB64, _ := strconv.ParseUint(fields[2], 10, 64)
		usedKiB64, _ := strconv.ParseUint(fields[3], 10, 64)
		usage := humanSwap(sizeKiB64, usedKiB64)

		if strings.HasPrefix(filename, "/dev/zram") {
			zram = usage
		} else if filename == "/swapfile" {
			swapfile = usage
		}
	}
	return zram, swapfile
}

func getDiskUsage(path string) string {
	d, err := disk.Usage(path)
	if err != nil {
		return "н/д"
	}
	return fmt.Sprintf("%.2f%% (%.1f GB / %.1f GB)",
		d.UsedPercent, float64(d.Used)/1e9, float64(d.Total)/1e9)
}

func getDiskFree(path string) string {
	d, err := disk.Usage(path)
	if err != nil {
		return "н/д"
	}
	return formatBytesIEC(d.Free)
}

func getTemperature() string {
	sensors, err := host.SensorsTemperatures()
	if err != nil {
		return "н/д"
	}
	for _, s := range sensors {
		if s.SensorKey == "cpu-thermal" || s.Temperature > 20 {
			return fmt.Sprintf("%.1f°C", s.Temperature)
		}
	}
	return "н/д"
}

func getLoadAvg() string {
	loadStat, err := load.Avg()
	if err != nil {
		return "н/д"
	}
	return fmt.Sprintf("%.2f / %.2f / %.2f (1/5/15 хв)",
		loadStat.Load1, loadStat.Load5, loadStat.Load15)
}

func getUptime() string {
	seconds, err := host.Uptime()
	if err != nil {
		return "н/д"
	}
	return formatUptime(seconds)
}

func getBootTime() string {
	seconds, err := host.BootTime()
	if err != nil {
		return "н/д"
	}
	return time.Unix(int64(seconds), 0).Format("2006-01-02 15:04:05")
}

func getProcessCount() int {
	procs, err := process.Processes()
	if err != nil {
		return 0
	}
	return len(procs)
}

func getLoggedInUsersCount() int {
	users, err := host.Users()
	if err != nil {
		return 0
	}
	return len(users)
}

func isProcessRunning(name string) string {
	procs, err := process.Processes()
	if err != nil {
		return "н/д"
	}
	for _, p := range procs {
		n, _ := p.Name()
		if n == name {
			return "✅"
		}
	}
	return "❌"
}

func getNetworkTotals() (string, string) {
	stats, err := gonet.IOCounters(false)
	if err != nil || len(stats) == 0 {
		return "н/д", "н/д"
	}

	return formatBytesIEC(stats[0].BytesRecv), formatBytesIEC(stats[0].BytesSent)
}

func getNetworkSpeed() (string, string) {
	stats, err := gonet.IOCounters(false)
	if err != nil || len(stats) == 0 {
		return "н/д", "н/д"
	}

	now := time.Now()
	currRx := stats[0].BytesRecv
	currTx := stats[0].BytesSent

	netSpeedMu.Lock()
	defer netSpeedMu.Unlock()

	if prevNetSampleAt.IsZero() {
		prevNetRxBytes = currRx
		prevNetTxBytes = currTx
		prevNetSampleAt = now
		return "н/д", "н/д"
	}

	elapsed := now.Sub(prevNetSampleAt).Seconds()
	if elapsed <= 0 {
		return "н/д", "н/д"
	}

	var rxBps float64
	var txBps float64

	if currRx >= prevNetRxBytes {
		rxBps = float64(currRx-prevNetRxBytes) / elapsed
	}
	if currTx >= prevNetTxBytes {
		txBps = float64(currTx-prevNetTxBytes) / elapsed
	}

	prevNetRxBytes = currRx
	prevNetTxBytes = currTx
	prevNetSampleAt = now

	return formatBytesPerSecondIEC(rxBps), formatBytesPerSecondIEC(txBps)
}

func getSystemdServiceStatus(service string) string {
	out, err := runCommand(2, "systemctl", "is-active", service)
	if err != nil {
		return "❌"
	}

	switch out {
	case "active":
		return "✅"
	case "activating":
		return "🟡"
	default:
		return "❌"
	}
}

func getFirstActiveServiceStatus(services ...string) string {
	for _, service := range services {
		status := getSystemdServiceStatus(service)
		if status == "✅" || status == "🟡" {
			return status
		}
	}
	return "❌"
}

func getServicesBlock() string {
	jellyfin := getSystemdServiceStatus("jellyfin")
	qbittorrent := getFirstActiveServiceStatus("qbittorrent-nox", "qbittorrent", "qbittorrent.service", "qbittorrent-nox.service")
	sonarr := getSystemdServiceStatus("sonarr")
	radarr := getSystemdServiceStatus("radarr")
	prowlarr := getSystemdServiceStatus("prowlarr")
	fail2ban := getSystemdServiceStatus("fail2ban")

	return fmt.Sprintf(
		`🧩 *Сервіси:*
• Jellyfin: %s
• qBittorrent: %s
• Sonarr: %s
• Radarr: %s
• Prowlarr: %s
• Fail2Ban: %s`,
		jellyfin,
		qbittorrent,
		sonarr,
		radarr,
		prowlarr,
		fail2ban,
	)
}

func parseThrottledFlags(hexValue uint64) string {
	var issues []string

	if hexValue&0x1 != 0 {
		issues = append(issues, "зараз undervoltage")
	}
	if hexValue&0x2 != 0 {
		issues = append(issues, "зараз arm freq capped")
	}
	if hexValue&0x4 != 0 {
		issues = append(issues, "зараз throttled")
	}
	if hexValue&0x8 != 0 {
		issues = append(issues, "зараз soft temp limit")
	}
	if hexValue&0x10000 != 0 {
		issues = append(issues, "був undervoltage")
	}
	if hexValue&0x20000 != 0 {
		issues = append(issues, "було arm freq capped")
	}
	if hexValue&0x40000 != 0 {
		issues = append(issues, "був throttling")
	}
	if hexValue&0x80000 != 0 {
		issues = append(issues, "був soft temp limit")
	}

	if len(issues) == 0 {
		return "немає"
	}

	return strings.Join(issues, ", ")
}

func getThrottledStatus() string {
	out, err := runCommand(2, "vcgencmd", "get_throttled")
	if err != nil {
		return "н/д"
	}

	re := regexp.MustCompile(`throttled=0x([0-9a-fA-F]+)`)
	m := re.FindStringSubmatch(out)
	if len(m) < 2 {
		return "н/д"
	}

	v, err := strconv.ParseUint(m[1], 16, 64)
	if err != nil {
		return "н/д"
	}

	return parseThrottledFlags(v)
}

func GetSystemMetrics() string {
	cpuUsage, cpuCount := getCPUUsage()
	cpuFreq := getCPUFreq()
	ramUsage := getMemoryUsage()
	zramUsage, swapfileUsage := getSwapDetailed()
	diskUsage := getDiskUsage("/")
	diskFree := getDiskFree("/")
	temp := getTemperature()
	loadAvg := getLoadAvg()
	uptime := getUptime()
	bootTime := getBootTime()
	procCount := getProcessCount()
	usersCount := getLoggedInUsersCount()
	publicIP := readExternalURL("https://api.ipify.org")
	ping := runCommandAndExtract(`time=([\d.]+) ms`, "ping", "-c", "1", "-w", "2", "8.8.8.8")
	netRx, netTx := getNetworkTotals()
	netRxSpeed, netTxSpeed := getNetworkSpeed()
	throttled := getThrottledStatus()
	servicesBlock := getServicesBlock()

	return fmt.Sprintf(`📊 *Метрики mykola-1*

🌡️ *Температура:* %s
🧠 *CPU:* %s (%d ядер)
⚙️ *CPU freq:* %s
📦 *Навантаження:* %s
💾 *RAM:* %s
💤 *ZRAM:* %s
💤 *SWAP file:* %s
🗄️ *SSD:* %s
🆓 *Вільно на SSD:* %s
📈 *Процесів:* %d
👤 *Користувачів онлайн:* %d
⏳ *Аптайм:* %s
🕓 *Завантажено о:* %s

🍓 *Raspberry Pi health:*
⚠️ *Throttled:* %s

🌐 *Мережа RX:* %s
🌐 *Мережа TX:* %s
⬇️ *Швидкість RX:* %s
⬆️ *Швидкість TX:* %s

🌍 *IP:* %s
📡 *Ping:* %s ms

%s`,
		temp,
		cpuUsage, cpuCount,
		cpuFreq,
		loadAvg,
		ramUsage,
		zramUsage,
		swapfileUsage,
		diskUsage,
		diskFree,
		procCount,
		usersCount,
		uptime,
		bootTime,
		throttled,
		netRx,
		netTx,
		netRxSpeed,
		netTxSpeed,
		publicIP,
		ping,
		servicesBlock,
	)
}
