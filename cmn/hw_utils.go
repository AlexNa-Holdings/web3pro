package cmn

// Vendor and Product IDs
const (
	VID_Trezor1 = 0x534c
	VID_Trezor2 = 0x1209
	VID_Ledger  = 0x2c97
)

const (
	TypeT1Hid    = 0
	TypeT1Webusb = 1

	TypeT1WebusbBoot    = 2
	TypeT2              = 3
	TypeT2Boot          = 4
	TypeEmulator        = 5
	ProductT2Bootloader = 0x53C0
	ProductT2Firmware   = 0x53C1
	LedgerNanoS         = 0x0001
	LedgerNanoX         = 0x0004
	LedgerBlue          = 0x0000
	ProductT1Firmware   = 0x0001
)

func IsTrezor1(vid uint16, pid uint16) bool {
	return vid == VID_Trezor1 && pid == ProductT1Firmware
}

func IsTrezor2(vid uint16, pid uint16) bool {
	return vid == VID_Trezor2 && (pid == ProductT2Firmware || pid == ProductT2Bootloader)
}

func IsTrezor(vid uint16, pid uint16) bool {
	return IsTrezor1(vid, pid) || IsTrezor2(vid, pid)
}

func IsLedger(vid uint16, pid uint16) bool {
	return vid == VID_Ledger
}

func IsSupportedDevice(vid uint16, pid uint16) bool {
	return IsTrezor(vid, pid) || IsLedger(vid, pid)
}

func USBDeviceType(vid uint16, pid uint16) string {
	if IsTrezor(vid, pid) {
		return "trezor"
	}
	if IsLedger(vid, pid) {
		return "ledger"
	}
	return ""
}
