package utils

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	upsI2CBus  = 1
	upsI2CAddr = "0x2d"

	regChargeState = "0x02"
	regCommState   = "0x03"

	regVBUSVoltageLo = "0x10"
	regVBUSVoltageHi = "0x11"
	regVBUSCurrentLo = "0x12"
	regVBUSCurrentHi = "0x13"
	regVBUSPowerLo   = "0x14"
	regVBUSPowerHi   = "0x15"

	regBatteryVoltageLo = "0x20"
	regBatteryVoltageHi = "0x21"
	regBatteryCurrentLo = "0x22"
	regBatteryCurrentHi = "0x23"
	regBatteryPercentLo = "0x24"
	regBatteryPercentHi = "0x25"
	regRemainCapLo      = "0x26"
	regRemainCapHi      = "0x27"
	regRemainDisLo      = "0x28"
	regRemainDisHi      = "0x29"
	regRemainChgLo      = "0x2A"
	regRemainChgHi      = "0x2B"

	regCell1Lo = "0x30"
	regCell1Hi = "0x31"
	regCell2Lo = "0x32"
	regCell2Hi = "0x33"
	regCell3Lo = "0x34"
	regCell3Hi = "0x35"
	regCell4Lo = "0x36"
	regCell4Hi = "0x37"

	regFirmwareVersion = "0x50"

	minVBUSPresentMV = 5000
	upsCacheTTL      = 5 * time.Second
	i2cTimeoutSec    = 2
)

type UpsSnapshot struct {
	CommState   int
	ChargeState int

	VBUSVoltageMV int
	VBUSCurrentMA int
	VBUSPowerMW   int

	BatteryVoltageMV int
	BatteryCurrentMA int
	BatteryPercent   int
	RemainingMAh     int
	RemainDisMin     int
	RemainChgMin     int

	Cell1MV int
	Cell2MV int
	Cell3MV int
	Cell4MV int

	FirmwareVersion int
	ReadAt          time.Time
}

var (
	upsCacheMu       sync.Mutex
	upsCacheSnapshot *UpsSnapshot
	upsCacheErr      error
	upsCacheAt       time.Time
)

func ReadUpsSnapshot() (*UpsSnapshot, error) {
	upsCacheMu.Lock()
	defer upsCacheMu.Unlock()

	if upsCacheSnapshot != nil && time.Since(upsCacheAt) < upsCacheTTL {
		return upsCacheSnapshot, upsCacheErr
	}

	s, err := readUpsSnapshotRaw()
	upsCacheSnapshot = s
	upsCacheErr = err
	upsCacheAt = time.Now()

	return s, err
}

func InvalidateUpsCache() {
	upsCacheMu.Lock()
	defer upsCacheMu.Unlock()

	upsCacheSnapshot = nil
	upsCacheErr = nil
	upsCacheAt = time.Time{}
}

func readUpsSnapshotRaw() (*UpsSnapshot, error) {
	s := &UpsSnapshot{}
	var err error

	if s.CommState, err = readReg8(regCommState); err != nil {
		return nil, fmt.Errorf("не вдалося прочитати COMM state: %w", err)
	}
	if s.ChargeState, err = readReg8(regChargeState); err != nil {
		return nil, fmt.Errorf("не вдалося прочитати CHARGE state: %w", err)
	}

	if s.VBUSVoltageMV, err = readU16LE(regVBUSVoltageLo, regVBUSVoltageHi); err != nil {
		return nil, fmt.Errorf("не вдалося прочитати VBUS voltage: %w", err)
	}
	if s.VBUSCurrentMA, err = readU16LE(regVBUSCurrentLo, regVBUSCurrentHi); err != nil {
		return nil, fmt.Errorf("не вдалося прочитати VBUS current: %w", err)
	}
	if s.VBUSPowerMW, err = readU16LE(regVBUSPowerLo, regVBUSPowerHi); err != nil {
		return nil, fmt.Errorf("не вдалося прочитати VBUS power: %w", err)
	}

	if s.BatteryVoltageMV, err = readU16LE(regBatteryVoltageLo, regBatteryVoltageHi); err != nil {
		return nil, fmt.Errorf("не вдалося прочитати battery voltage: %w", err)
	}
	if s.BatteryCurrentMA, err = readS16LE(regBatteryCurrentLo, regBatteryCurrentHi); err != nil {
		return nil, fmt.Errorf("не вдалося прочитати battery current: %w", err)
	}
	if s.BatteryPercent, err = readU16LE(regBatteryPercentLo, regBatteryPercentHi); err != nil {
		return nil, fmt.Errorf("не вдалося прочитати battery percent: %w", err)
	}
	if s.RemainingMAh, err = readU16LE(regRemainCapLo, regRemainCapHi); err != nil {
		return nil, fmt.Errorf("не вдалося прочитати remaining capacity: %w", err)
	}
	if s.RemainDisMin, err = readU16LE(regRemainDisLo, regRemainDisHi); err != nil {
		return nil, fmt.Errorf("не вдалося прочитати remaining discharge time: %w", err)
	}
	if s.RemainChgMin, err = readU16LE(regRemainChgLo, regRemainChgHi); err != nil {
		return nil, fmt.Errorf("не вдалося прочитати remaining charge time: %w", err)
	}

	if s.Cell1MV, err = readU16LE(regCell1Lo, regCell1Hi); err != nil {
		return nil, fmt.Errorf("не вдалося прочитати cell1: %w", err)
	}
	if s.Cell2MV, err = readU16LE(regCell2Lo, regCell2Hi); err != nil {
		return nil, fmt.Errorf("не вдалося прочитати cell2: %w", err)
	}
	if s.Cell3MV, err = readU16LE(regCell3Lo, regCell3Hi); err != nil {
		return nil, fmt.Errorf("не вдалося прочитати cell3: %w", err)
	}
	if s.Cell4MV, err = readU16LE(regCell4Lo, regCell4Hi); err != nil {
		return nil, fmt.Errorf("не вдалося прочитати cell4: %w", err)
	}

	if s.FirmwareVersion, err = readReg8(regFirmwareVersion); err != nil {
		s.FirmwareVersion = -1
	}

	s.ReadAt = time.Now()

	return s, nil
}

func (s *UpsSnapshot) VBUSPresent() bool {
	return s.VBUSVoltageMV >= minVBUSPresentMV
}

func (s *UpsSnapshot) Charging() bool {
	return s.BatteryCurrentMA > 0
}

func (s *UpsSnapshot) Discharging() bool {
	return s.BatteryCurrentMA < 0
}

func (s *UpsSnapshot) StateEmoji() string {
	switch {
	case s.Charging():
		return "⚡️"
	case s.Discharging():
		return "🔋"
	default:
		return "🔌"
	}
}

func (s *UpsSnapshot) StateText() string {
	switch {
	case s.Charging():
		return "заряджається"
	case s.Discharging():
		return "розряджається"
	case s.VBUSPresent():
		return "підключено до живлення"
	default:
		return "стан невідомий"
	}
}

func (s *UpsSnapshot) ShortStateText() string {
	switch {
	case s.Charging():
		return "charging"
	case s.Discharging():
		return "discharging"
	case s.VBUSPresent():
		return "external_power"
	default:
		return "unknown"
	}
}

func (s *UpsSnapshot) PowerSourceText() string {
	switch {
	case s.VBUSPresent() && s.Charging():
		return "зовнішнє живлення + зарядка"
	case s.VBUSPresent():
		return "зовнішнє живлення"
	case s.Discharging():
		return "живлення від батареї"
	default:
		return "невідомо"
	}
}

func (s *UpsSnapshot) BatteryVoltageV() float64 {
	return float64(s.BatteryVoltageMV) / 1000.0
}

func (s *UpsSnapshot) BatteryCurrentA() float64 {
	return float64(s.BatteryCurrentMA) / 1000.0
}

func (s *UpsSnapshot) VBUSVoltageV() float64 {
	return float64(s.VBUSVoltageMV) / 1000.0
}

func (s *UpsSnapshot) VBUSCurrentA() float64 {
	return float64(s.VBUSCurrentMA) / 1000.0
}

func (s *UpsSnapshot) VBUSPowerW() float64 {
	return float64(s.VBUSPowerMW) / 1000.0
}

func (s *UpsSnapshot) CellMinMV() int {
	minV := s.Cell1MV
	for _, v := range []int{s.Cell2MV, s.Cell3MV, s.Cell4MV} {
		if v < minV {
			minV = v
		}
	}
	return minV
}

func (s *UpsSnapshot) CellMaxMV() int {
	maxV := s.Cell1MV
	for _, v := range []int{s.Cell2MV, s.Cell3MV, s.Cell4MV} {
		if v > maxV {
			maxV = v
		}
	}
	return maxV
}

func (s *UpsSnapshot) CellDeltaMV() int {
	return s.CellMaxMV() - s.CellMinMV()
}

func (s *UpsSnapshot) ETALabel() string {
	switch {
	case s.Discharging():
		return "Час роботи"
	case s.Charging():
		return "До повної зарядки"
	default:
		return "ETA"
	}
}

func (s *UpsSnapshot) ETAString() string {
	switch {
	case s.Discharging():
		return formatMinutesSmart(s.RemainDisMin)
	case s.Charging():
		return formatMinutesSmart(s.RemainChgMin)
	default:
		return "—"
	}
}

func (s *UpsSnapshot) BQ4050OK() bool {
	return s.CommState&0x01 == 0
}

func (s *UpsSnapshot) IP2368OK() bool {
	return s.CommState&0x02 == 0
}

func (s *UpsSnapshot) CommText() string {
	bq := "❌ помилка"
	if s.BQ4050OK() {
		bq = "✅ норма"
	}

	ip := "❌ помилка"
	if s.IP2368OK() {
		ip = "✅ норма"
	}

	return fmt.Sprintf("BQ4050: %s, IP2368: %s", bq, ip)
}

func (s *UpsSnapshot) IsFastCharging() bool {
	return s.ChargeState&0x80 != 0
}

func (s *UpsSnapshot) ChargePhase() string {
	phase := (s.ChargeState >> 4) & 0x07

	switch phase {
	case 0:
		return "очікування"
	case 1:
		return "попередній заряд"
	case 2:
		return "постійний струм"
	case 3:
		return "постійна напруга"
	case 4:
		return "заряд завершено"
	case 5:
		return "очікує зарядки"
	case 6:
		return "таймаут зарядки"
	default:
		return "невідомо"
	}
}

func (s *UpsSnapshot) ChargeDetailsText() string {
	if s.IsFastCharging() {
		return s.ChargePhase() + " (швидка зарядка)"
	}
	return s.ChargePhase()
}

func (s *UpsSnapshot) FirmwareText() string {
	if s.FirmwareVersion < 0 {
		return "н/д"
	}
	return fmt.Sprintf("0x%X", s.FirmwareVersion)
}

func GetUpsStatus() string {
	s, err := ReadUpsSnapshot()
	if err != nil {
		return "🔋 *UPS:* н/д\nПричина: " + err.Error()
	}

	return fmt.Sprintf(
		`%s *UPS HAT (E)*

📌 *Стан*
• Режим: %s
• Джерело: %s
• Зарядка: %s

⚡ *Живлення*
• VBUS: %.3f V / %.3f A / %.3f W
• Батарея: %.3f V / %.3f A

🔋 *Батарея*
• Заряд: %d%%
• Залишок: %d mAh
• %s: %s

🔬 *Банки*
• Банка 1: %d mV
• Банка 2: %d mV
• Банка 3: %d mV
• Банка 4: %d mV
• Дельта: %d mV

🧩 *Службова інформація*
• Комунікація: %s
• Прошивка: %s`,
		s.StateEmoji(),
		s.StateText(),
		s.PowerSourceText(),
		s.ChargeDetailsText(),
		s.VBUSVoltageV(), s.VBUSCurrentA(), s.VBUSPowerW(),
		s.BatteryVoltageV(), s.BatteryCurrentA(),
		s.BatteryPercent,
		s.RemainingMAh,
		s.ETALabel(), s.ETAString(),
		s.Cell1MV,
		s.Cell2MV,
		s.Cell3MV,
		s.Cell4MV,
		s.CellDeltaMV(),
		s.CommText(),
		s.FirmwareText(),
	)
}

func GetUpsShortStatus() string {
	s, err := ReadUpsSnapshot()
	if err != nil {
		return "UPS: н/д"
	}

	etaLabel := "ETA"
	if s.Discharging() {
		etaLabel = "залишилось"
	} else if s.Charging() {
		etaLabel = "до повної"
	}

	return fmt.Sprintf(
		"UPS: %s, %d%%, джерело=%s, VBUS=%.3fV/%.3fA/%.3fW, батарея=%.3fV/%.3fA, %s=%s, дельта=%dmV",
		s.StateText(),
		s.BatteryPercent,
		s.PowerSourceText(),
		s.VBUSVoltageV(), s.VBUSCurrentA(), s.VBUSPowerW(),
		s.BatteryVoltageV(), s.BatteryCurrentA(),
		etaLabel, s.ETAString(),
		s.CellDeltaMV(),
	)
}

func readReg8(reg string) (int, error) {
	cmd := exec.Command(
		"timeout",
		strconv.Itoa(i2cTimeoutSec),
		"i2cget",
		"-y",
		strconv.Itoa(upsI2CBus),
		upsI2CAddr,
		reg,
	)

	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	val := strings.TrimSpace(string(out))
	n, err := strconv.ParseInt(strings.TrimPrefix(val, "0x"), 16, 32)
	if err != nil {
		return 0, fmt.Errorf("не вдалося розпарсити %q: %w", val, err)
	}

	return int(n), nil
}

func readU16LE(regLo, regHi string) (int, error) {
	lo, err := readReg8(regLo)
	if err != nil {
		return 0, err
	}
	hi, err := readReg8(regHi)
	if err != nil {
		return 0, err
	}
	return (hi << 8) | lo, nil
}

func readS16LE(regLo, regHi string) (int, error) {
	v, err := readU16LE(regLo, regHi)
	if err != nil {
		return 0, err
	}
	if v >= 32768 {
		return v - 65536, nil
	}
	return v, nil
}

func formatMinutesSmart(mins int) string {
	if mins <= 0 || mins >= 65535 {
		return "—"
	}

	d := mins / 1440
	h := (mins % 1440) / 60
	m := mins % 60

	switch {
	case d > 0:
		return fmt.Sprintf("%d д %d год %d хв", d, h, m)
	case h > 0:
		return fmt.Sprintf("%d год %d хв", h, m)
	default:
		return fmt.Sprintf("%d хв", m)
	}
}
