package utils

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
)

// ---------- helpers: cgroup v2 readers ----------

func readFirstLine(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	if sc.Scan() {
		return strings.TrimSpace(sc.Text()), nil
	}
	return "", sc.Err()
}

func parseCPUSetList(list string) int {
	// приклади: "0-1", "0-1,3", "2,4-5"
	count := 0
	for _, part := range strings.Split(list, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			b := strings.SplitN(part, "-", 2)
			start, _ := strconv.Atoi(b[0])
			end, _ := strconv.Atoi(b[1])
			count += end - start + 1
		} else {
			count++
		}
	}
	return count
}

func getCPUAlloc() int {
	// 1) cpu.max: "<quota> <period>" або "max <period>"
	if s, err := readFirstLine("/sys/fs/cgroup/cpu.max"); err == nil {
		parts := strings.Fields(s)
		if len(parts) == 2 {
			if parts[0] != "max" {
				quota, _ := strconv.ParseFloat(parts[0], 64)
				period, _ := strconv.ParseFloat(parts[1], 64) // типовий period 100000
				if period > 0 {
					vcpu := int((quota / period) + 0.9999) // ceil
					if vcpu < 1 {
						vcpu = 1
					}
					return vcpu
				}
			}
		}
	}
	// 2) cpuset.cpus.effective як запасний варіант
	if s, err := readFirstLine("/sys/fs/cgroup/cpuset.cpus.effective"); err == nil && s != "" {
		if n := parseCPUSetList(s); n > 0 {
			return n
		}
	}
	// 3) fallback: скільки бачить ядeр (може показати всі, але краще ніж 0)
	if n, err := cpu.Counts(true); err == nil && n > 0 {
		return n
	}
	return 0
}

func getMemLimitBytes() (limit uint64, unlimited bool) {
	// memory.max: або число у байтах, або "max"
	if s, err := readFirstLine("/sys/fs/cgroup/memory.max"); err == nil {
		if s == "max" {
			return 0, true
		}
		if v, err2 := strconv.ParseUint(s, 10, 64); err2 == nil {
			return v, false
		}
	}
	return 0, true
}

func getMemCurrentBytes() (used uint64, err error) {
	if s, err := readFirstLine("/sys/fs/cgroup/memory.current"); err == nil {
		return strconv.ParseUint(s, 10, 64)
	}
	return 0, fmt.Errorf("memory.current not found")
}

// ---------- simple metrics (container-aware) ----------

func getCPUSimple() (pct string, allocCores int) {
	// Використання CPU (миттєве) + скільки vCPU виділено за cgroup
	percents, _ := cpu.Percent(0, false)
	p := "н/д"
	if len(percents) > 0 {
		p = fmt.Sprintf("%.1f%%", percents[0])
	}
	return p, getCPUAlloc()
}

func bytesToGiB(b uint64) float64 {
	return float64(b) / (1024.0 * 1024.0 * 1024.0)
}

func getRAMSimple() (usedGiB, limitGiB float64, pct float64) {
	usedBytes, err := getMemCurrentBytes()
	limitBytes, unlim := getMemLimitBytes()
	if err != nil {
		return 0, 0, 0
	}
	usedGiB = bytesToGiB(usedBytes)
	if unlim || limitBytes == 0 {
		// якщо ліміт не виставлено — покажемо як "н/д"
		return usedGiB, 0, 0
	}
	limitGiB = bytesToGiB(limitBytes)
	pct = (usedGiB / limitGiB) * 100.0
	return usedGiB, limitGiB, pct
}

func getDiskSimple(path string) (usedGB, totalGB float64, pct float64, err error) {
	du, err := disk.Usage(path)
	if err != nil {
		return 0, 0, 0, err
	}
	return float64(du.Used) / 1e9, float64(du.Total) / 1e9, du.UsedPercent, nil
}

// GetSimpleMetrics без перевірки логіну користувача
func GetSimpleMetrics() string {
	cpuPct, cpuCores := getCPUSimple()

	ramUsed, ramLimit, ramPct := getRAMSimple()
	ramLine := "💾 *RAM:* н/д"
	if ramLimit > 0 {
		ramLine = fmt.Sprintf("💾 *RAM:* %.2f%% (%.1f GiB / %.1f GiB)", ramPct, ramUsed, ramLimit)
	} else {
		ramLine = fmt.Sprintf("💾 *RAM:* %.1f GiB / н/д (ліміт не виставлено?)", ramUsed)
	}

	diskUsed, diskTotal, diskPct, _ := getDiskSimple("/")
	diskLine := fmt.Sprintf("🗄️ *Disk (/):* %.2f%% (%.1f GB / %.1f GB)", diskPct, diskUsed, diskTotal)

	// покажемо, де ми беремо cgroup (на випадок кастомних шляхів)
	cgRoot := "/sys/fs/cgroup"
	if _, err := os.Stat(filepath.Join(cgRoot, "cpu.max")); err != nil {
		cgRoot = "(cgroup v2 не знайдено?)"
	}

	return fmt.Sprintf(`📋 *Прості метрики mykola-1 (контейнер)*

🧠 *CPU:* %s (виділено ~%d vCPU)
%s
%s
`, cpuPct, cpuCores, ramLine, diskLine)
}
