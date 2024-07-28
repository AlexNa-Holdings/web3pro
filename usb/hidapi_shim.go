//go:build linux || freebsd || openbsd
// +build linux freebsd openbsd

// shim for linux and freebsd so that trezord.go builds

package usb

const HIDUse = false

type HIDAPI struct {
}

func InitHIDAPI() (*HIDAPI, error) {
	return &HIDAPI{}, nil
}

func (b *HIDAPI) Enumerate() ([]USBInfo, error) {
	panic("not implemented for linux and freebsd")
}

func (b *HIDAPI) Has(path string) bool {
	panic("not implemented for linux and freebsd")
}

func (b *HIDAPI) Connect(path string, debug bool, reset bool) (USBDevice, error) {
	return &HID{}, nil
}

type HID struct {
}

func (d *HID) Close(disconnected bool) error {
	panic("not implemented for linux and freebsd")
}

func (d *HID) Write(buf []byte) (int, error) {
	panic("not implemented for linux and freebsd")
}

func (d *HID) Read(buf []byte) (int, error) {
	panic("not implemented for linux and freebsd")
}

func (b *HIDAPI) Close() {
	panic("not implemented for linux and freebsd")
}
