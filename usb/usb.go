package usb

import "github.com/karalabe/usb"

func List() ([]usb.DeviceInfo, error) {

	return usb.EnumerateHid(0, 0)
}
