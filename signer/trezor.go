package signer

import (
	"errors"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/address"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/usb_support"
	"github.com/karalabe/usb"
	"github.com/rs/zerolog/log"
)

type TrezorDriver struct {
	*Signer
	usb.DeviceInfo
	usb.Device
}

func NewTrezorDriver(s *Signer) (TrezorDriver, error) {

	if s.Type != "trezor" {
		return TrezorDriver{}, errors.New("invalid signer type")
	}

	return TrezorDriver{
		Signer: s,
	}, nil
}

func (d TrezorDriver) IsConnected() bool {
	devices, err := usb_support.List()
	if err != nil {
		log.Error().Msgf("Error listing USB devices", "err", err)
		return false
	}

	sns := []string{d.SN}
	for _, c := range d.Copies {
		sns = append(sns, c.SN)
	}

	for _, device := range devices {
		if device.Product == "TREZOR" && cmn.IsInArray(sns, device.Serial) {
			return true
		}
	}
	return false
}

func (d TrezorDriver) FindDeviceInfo() (usb.DeviceInfo, error) {
	devices, err := usb_support.List()
	if err != nil {
		log.Error().Msgf("Error listing USB devices", "err", err)
		return usb.DeviceInfo{}, err
	}

	sns := []string{d.SN}
	names := []string{d.Name}
	for _, c := range d.Copies {
		sns = append(sns, c.SN)
		names = append(names, c.Name)
	}

	for _, info := range devices {
		if info.Product == "TREZOR" && cmn.IsInArray(sns, info.Serial) {

			return info, nil
		}
	}

	cmn.Hail(&cmn.HailRequest{
		Title: "Connect Trezor",
		Template: `<c><w>
Connect your Trezor device and unlock it.
<b><u>` + strings.Join(names, ", ") + `</u></b>

<button text:Ok tip:"create wallet">  <button text:Cancel>
`,
	})

	return usb.DeviceInfo{}, errors.New("device not found")
}

func (d TrezorDriver) GetAddresses(path_format string, start_from int, count int) ([]address.Address, error) {
	var err error
	addresses := []address.Address{}

	d.DeviceInfo, err = d.FindDeviceInfo()
	if err != nil {
		log.Debug().Msgf("Error finding device: %s", err)
		return addresses, err
	}

	return addresses, nil

}
