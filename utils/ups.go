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

	regFullCapLo = "0x2C"
	regFullCapHi = "0x2D"

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
	FullCapacityMAh  int
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

func readUpsSnapshotRaw() (*UpsSnapshot, error) {
	s := &UpsSnapshot{}
	var err error

	if s.CommState, err = readReg8(regCommState); err != nil {
		return nil, err
	}
	if s.ChargeState, err = readReg8(regChargeState); err != nil {
		return nil, err
	}

	if s.VBUSVoltageMV, err = readU16LE(regVBUSVoltageLo, regVBUSVoltageHi); err != nil {
		return nil, err
	}
	if s.VBUSCurrentMA, err = readU16LE(regVBUSCurrentLo, regVBUSCurrentHi); err != nil {
		return nil, err
	}
	if s.VBUSPowerMW, err = readU16LE(regVBUSPowerLo, regVBUSPowerHi); err != nil {
		return nil, err
	}

	if s.BatteryVoltageMV, err = readU16LE(regBatteryVoltageLo, regBatteryVoltageHi); err != nil {
		return nil, err
	}
	if s.BatteryCurrentMA, err = readS16LE(regBatteryCurrentLo, regBatteryCurrentHi); err != nil {
		return nil, err
	}
	if s.BatteryPercent, err = readU16LE(regBatteryPercentLo, regBatteryPercentHi); err != nil {
		return nil, err
	}
	if s.RemainingMAh, err = readU16LE(regRemainCapLo, regRemainCapHi); err != nil {
		return nil, err
	}
	if s.RemainDisMin, err = readU16LE(regRemainDisLo, regRemainDisHi); err != nil {
		return nil, err
	}
	if s.RemainChgMin, err = readU16LE(regRemainChgLo, regRemainChgHi); err != nil {
		return nil, err
	}
	if s.FullCapacityMAh, err = readU16LE(regFullCapLo, regFullCapHi); err != nil {
		s.FullCapacityMAh = 0
	}

	if s.Cell1MV, err = readU16LE(regCell1Lo, regCell1Hi); err != nil {
		return nil, err
	}
	if s.Cell2MV, err = readU16LE(regCell2Lo, regCell2Hi); err != nil {
		return nil, err
	}
	if s.Cell3MV, err = readU16LE(regCell3Lo, regCell3Hi); err != nil {
		return nil, err
	}
	if s.Cell4MV, err = readU16LE(regCell4Lo, regCell4Hi); err != nil {
		return nil, err
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

func (s *UpsSnapshot) IsFastCharging() bool {
	return s.ChargeState&0x80 != 0
}

func (s *UpsSnapshot) ChargeDetailsText() string {
	if s.IsFastCharging() {
		return s.ChargePhase() + " (швидка)"
	}
	return s.ChargePhase()
}

func (s *UpsSnapshot) BatteryVoltageV() float64 {
	return float64(s.BatteryVoltageMV) / 1000
}

func (s *UpsSnapshot) BatteryCurrentA() float64 {
	return float64(s.BatteryCurrentMA) / 1000
}

func (s *UpsSnapshot) VBUSVoltageV() float64 {
	return float64(s.VBUSVoltageMV) / 1000
}

func (s *UpsSnapshot) VBUSCurrentA() float64 {
	return float64(s.VBUSCurrentMA) / 1000
}

func (s *UpsSnapshot) VBUSPowerW() float64 {
	return float64(s.VBUSPowerMW) / 1000
}

func (s *UpsSnapshot) CellDeltaMV() int {
	minV := s.Cell1MV
	maxV := s.Cell1MV

	for _, v := range []int{s.Cell2MV, s.Cell3MV, s.Cell4MV} {
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
	}

	return maxV - minV
}

func (s *UpsSnapshot) CellDeltaText() string {
	d := s.CellDeltaMV()

	switch {
	case d < 50:
		return fmt.Sprintf("%d mV (ідеально)", d)
	case d < 100:
		return fmt.Sprintf("%d mV (норма)", d)
	case d < 200:
		return fmt.Sprintf("%d mV (є розбаланс)", d)
	default:
		return fmt.Sprintf("%d mV (⚠️ проблема)", d)
	}
}

func (s *UpsSnapshot) ETALabel() string {
	if s.Discharging() {
		return "Час роботи"
	}
	if s.Charging() {
		return "До повної зарядки"
	}
	return "Стан"
}

func (s *UpsSnapshot) ETAString() string {
	if s.Discharging() {
		return formatMinutesSmart(s.RemainDisMin)
	}
	if s.Charging() {
		return formatMinutesSmart(s.RemainChgMin)
	}
	return "немає активного процесу"
}

func (s *UpsSnapshot) BQ4050OK() bool {
	return s.CommState&0x01 == 0
}

func (s *UpsSnapshot) IP2368OK() bool {
	return s.CommState&0x02 == 0
}

func (s *UpsSnapshot) CommText() string {
	bq := "активний"
	if !s.BQ4050OK() {
		bq = "не активний"
	}

	ip := "активний"
	if !s.IP2368OK() {
		if !s.VBUSPresent() {
			ip = "вимкнений"
		} else {
			ip = "не активний"
		}
	}

	return fmt.Sprintf("BQ4050: %s, IP2368: %s", bq, ip)
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
		return "UPS: н/д"
	}

	vbusV := s.VBUSVoltageV()
	vbusA := s.VBUSCurrentA()
	vbusW := s.VBUSPowerW()
	batV := s.BatteryVoltageV()
	batA := s.BatteryCurrentA()

	chargeText := "—"
	if s.VBUSPresent() {
		chargeText = s.ChargeDetailsText()
	}

	capacityText := fmt.Sprintf("%d mAh", s.RemainingMAh)
	if s.FullCapacityMAh > 0 {
		capacityText = fmt.Sprintf("%d / %d mAh", s.RemainingMAh, s.FullCapacityMAh)
	}

	return fmt.Sprintf(
		`%s UPS HAT (E)

📌 Стан
• Режим: %s
• Джерело: %s
• Зарядка: %s

⚡ Живлення
• VBUS: %.3f V / %.3f A / %.3f W
• Батарея: %.3f V / %.3f A

🔋 Батарея
• Заряд: %d%% (%s)
• %s: %s

🔬 Банки
• 1: %d mV | 2: %d mV | 3: %d mV | 4: %d mV
• Дельта: %s

🧩 Система
• Комунікація: %s
• Прошивка: %s`,
		s.StateEmoji(),
		s.StateText(),
		s.PowerSourceText(),
		chargeText,
		vbusV, vbusA, vbusW,
		batV, batA,
		s.BatteryPercent,
		capacityText,
		s.ETALabel(), s.ETAString(),
		s.Cell1MV, s.Cell2MV, s.Cell3MV, s.Cell4MV,
		s.CellDeltaText(),
		s.CommText(),
		s.FirmwareText(),
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
		return 0, err
	}

	return int(n), nil
}

func readU16LE(lo, hi string) (int, error) {
	l, err := readReg8(lo)
	if err != nil {
		return 0, err
	}
	h, err := readReg8(hi)
	if err != nil {
		return 0, err
	}
	return (h << 8) | l, nil
}

func readS16LE(lo, hi string) (int, error) {
	v, err := readU16LE(lo, hi)
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

	if d > 0 {
		return fmt.Sprintf("%d д %d год %d хв", d, h, m)
	}
	if h > 0 {
		return fmt.Sprintf("%d год %d хв", h, m)
	}
	return fmt.Sprintf("%d хв", m)
}
