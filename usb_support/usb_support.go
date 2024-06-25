package usb_support

import (
	"github.com/karalabe/usb"
)

func List() ([]usb.DeviceInfo, error) {

	return usb.EnumerateHid(0, 0)
}

func GetSN(info usb.DeviceInfo) (string, error) {

	switch info.Manufacturer {
	case "Ledger":
		// TO DO
		return "12345", nil
	}
	return info.Serial, nil
}
