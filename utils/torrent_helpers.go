package utils

import (
	"fmt"
	"strings"
)

func IsUploadingState(state string) bool {
	switch state {
	case "uploading", "forcedUP", "stalledUP", "queuedUP":
		return true
	default:
		return false
	}
}

func MapState(state string) string {
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

func FormatSpeedOrDash(b int64) string {
	if b <= 0 {
		return "—"
	}
	return FormatSpeed(b)
}

func FormatSpeed(b int64) string {
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

func FormatETA(seconds int64) string {
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

func ProgressBar(percent int, width int) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	filled := percent * width / 100

	return "[" + strings.Repeat("■", filled) + strings.Repeat("□", width-filled) + "]"
}

func Truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "..."
}

func EffectiveSize(size int64, totalSize int64) int64 {
	if totalSize > 0 {
		return totalSize
	}
	return size
}

func FormatBytesIECInt64(b int64) string {
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
