package usb

import (
	"errors"
)

type USB struct {
	buses []USBBus
}

func Init(buses ...USBBus) *USB {
	return &USB{
		buses: buses,
	}
}

func (b *USB) Has(path string) bool {
	for _, b := range b.buses {
		if b.Has(path) {
			return true
		}
	}
	return false
}

func (b *USB) Enumerate() ([]USBInfo, error) {
	infos := make([]USBInfo, 0, len(b.buses))

	for _, b := range b.buses {
		l, err := b.Enumerate()
		if err != nil {
			return nil, err
		}
		infos = append(infos, l...)
	}
	return infos, nil
}

func (b *USB) Connect(path string, debug bool, reset bool) (USBDevice, error) {
	for _, b := range b.buses {
		if b.Has(path) {
			return b.Connect(path, debug, reset)
		}
	}
	return nil, ErrNotFound
}

func (b *USB) Close() {
	for _, b := range b.buses {
		b.Close()
	}
}

var ErrNotFound = errors.New("device not found")
var errDisconnect = errors.New("device disconnected during action")
var errClosedDevice = errors.New("closed device")
var errNotDebug = errors.New("not debug link")
