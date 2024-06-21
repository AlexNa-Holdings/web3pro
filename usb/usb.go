package usb

import "github.com/karalabe/usb"

func List() ([]usb.DeviceInfo, error) {

	return usb.EnumerateHid(0, 0)
}

func GetSN(device usb.DeviceInfo) (string, error) {

	switch device.Manufacturer {
	case "Ledger":
		// TO DO
		return "12345", nil
	}

	return device.Serial, nil
}
