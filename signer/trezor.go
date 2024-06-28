package signer

import (
	"errors"

	"github.com/AlexNa-Holdings/web3pro/address"
)

// // ErrTrezorPINNeeded is returned if opening the trezor requires a PIN code. In
// // this case, the calling application should display a pinpad and send back the
// // encoded passphrase.
// var ErrTrezorPINNeeded = errors.New("trezor: pin needed")

// // ErrTrezorPassphraseNeeded is returned if opening the trezor requires a passphrase
// var ErrTrezorPassphraseNeeded = errors.New("trezor: passphrase needed")

// // errTrezorReplyInvalidHeader is the error message returned by a Trezor data exchange
// // if the device replies with a mismatching header. This usually means the device
// // is in browser mode.
// var errTrezorReplyInvalidHeader = errors.New("trezor: invalid reply header")

type TrezorDriver struct {
	*Signer
	// usb.DeviceInfo
	// usb.Device

	session_id         string
	password           string
	password_confirmed bool
	version            [3]uint32
	label              string

	failure error // Any failure that would make the device unusable
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
	// devices, err := core.List()
	// if err != nil {
	// 	log.Error().Msgf("Error listing USB devices", "err", err)
	// 	return false
	// }

	// sns := []string{d.SN}
	// for _, c := range d.Copies {
	// 	sns = append(sns, c.SN)
	// }

	// for _, device := range devices {
	// 	if device.Product == "TREZOR" && cmn.IsInArray(sns, device.Serial) {
	// 		return true
	// 	}
	// }
	return false
}

// func (d TrezorDriver) FindDeviceInfo() (usb.DeviceInfo, error) {
// sns := []string{d.SN}
// 	names := []string{d.Name}
// 	for _, c := range d.Copies {
// 		sns = append(sns, c.SN)
// 		names = append(names, c.Name)
// 	}

// 	devices, err := core.List()
// 	if err != nil {
// 		log.Error().Msgf("FindDeviceInfo: Error listing USB devices", "err", err)
// 		return usb.DeviceInfo{}, err
// 	}

// 	for _, info := range devices {
// 		if info.Product == "TREZOR" && cmn.IsInArray(sns, info.Serial) {

// 			return info, nil
// 		}
// 	}

// 	var ret usb.DeviceInfo

// 	cmn.HailAndWait(&cmn.HailRequest{
// 		Title: "Connect Trezor",
// 		Template: `<c><w>
// Connect your Trezor device and unlock it.
// <b><u>` + strings.Join(names, ", ") + `</u></b>

// <button text:Cancel>
// `,
// 		OnTick: func(hail *cmn.HailRequest, tick int) {
// 			if tick%5 == 0 { // once every n ticks
// 				devices, err := core.List()
// 				if err != nil {
// 					log.Error().Msgf("FindDeviceInfo: Error listing USB devices", "err", err)
// 					return
// 				}
// 				for _, info := range devices {
// 					if info.Product == "TREZOR" && cmn.IsInArray(sns, info.Serial) {
// 						ret = info
// 						log.Trace().Msgf("FindDeviceInfo: Trezor device connected %v", ret)
// 						hail.Close()
// 						break
// 					}
// 				}
// 			}
// 		},
// 		OnCancel: func(hr *cmn.HailRequest) {
// 			log.Debug().Msgf("FindDeviceInfo: User clicked Cancel")
// 		},
// 	})

// 	log.Debug().Msgf("FindDeviceInfo: After Trezor device found %v", ret)

// 	if ret.Path == "" {
// 		log.Error().Msgf("FindDeviceInfo: Trezor device not found")
// 		return ret, errors.New("device not found")
// 	}

// return ret, nil
// }

func (d TrezorDriver) GetAddresses(path_format string, start_from int, count int) ([]address.Address, error) {
	// var err error
	addresses := []address.Address{}

	// if err = d.Open(); err != nil {
	// 	log.Error().Msgf("GetAddresses: Error opening Trezor device", "err", err)
	// 	return addresses, err
	// }

	// for i := start_from; i < start_from+count; i++ {
	// 	path := fmt.Sprintf(path_format, i)
	// 	derivationPath, err := accounts.ParseDerivationPath(path)
	// 	if err != nil {
	// 		return addresses, err
	// 	}

	// 	addr, err := d.trezorDerive(derivationPath)
	// 	if err != nil {
	// 		return addresses, err
	// 	}

	// 	addresses = append(addresses, address.Address{
	// 		Name:    "",
	// 		Address: addr,
	// 		Path:    path,
	// 		Signer:  d.Signer.Name,
	// 	})
	// }

	return addresses, nil

}

func (d *TrezorDriver) Open() error {
	// var err error

	// log.Trace().Msgf("Open: %s", d.DeviceInfo.Path)

	// if d.Device == nil {
	// 	d.failure = nil

	// 	log.Trace().Msgf("Opening Trezor device %s", d.DeviceInfo.Product)

	// 	d.DeviceInfo, err = d.FindDeviceInfo()
	// 	if err != nil {
	// 		return err
	// 	}

	// 	log.Trace().Msgf("Trezor device info: %v", d.DeviceInfo)

	// 	d.Device, err = d.DeviceInfo.Open()
	// 	if err != nil {
	// 		return err
	// 	}

	// 	log.Trace().Msgf("Trezor device opened %v", d.Device)

	// 	// Initialize a connection to the device
	// 	features := new(trezor.Features)
	// 	if _, err := d.trezorExchange(&trezor.Initialize{}, features); err != nil {
	// 		log.Error().Msgf("Opne: Error initializing Trezor device", "err", err)
	// 		return err
	// 	}
	// 	d.version = [3]uint32{features.GetMajorVersion(), features.GetMinorVersion(), features.GetPatchVersion()}
	// 	d.label = features.GetLabel()

	// 	log.Trace().Msgf("Trezor initialized version: %v label: %s", d.version, d.label)
	// }

	// // Do a manual ping, forcing the device to ask for its PIN and Passphrase
	// askPin := true
	// askPassphrase := true
	// res, err := d.trezorExchange(&trezor.Ping{PinProtection: &askPin, PassphraseProtection: &askPassphrase}, new(trezor.PinMatrixRequest), new(trezor.PassphraseRequest), new(trezor.Success))
	// if err != nil {
	// 	return err
	// }
	// Only return the PIN request if the device wasn't unlocked until now

	// log.Debug().Msgf("res: %v", res)

	// switch res {
	// case 0:
	// 	d.pinwait = true
	// 	return ErrTrezorPINNeeded
	// case 1:
	// 	d.pinwait = false
	// 	d.passphrasewait = true
	// 	return ErrTrezorPassphraseNeeded
	// case 2:
	// 	return nil // responded with trezor.Success
	// }

	// // If phase 1 is requested, init the connection and wait for user callback
	// if passphrase == "" && !d.passphrasewait {
	// 	// If we're already waiting for a PIN entry, insta-return
	// 	if d.pinwait {
	// 		return ErrTrezorPINNeeded
	// 	}

	// 	// Do a manual ping, forcing the device to ask for its PIN and Passphrase
	// 	askPin := true
	// 	askPassphrase := true
	// 	res, err := d.trezorExchange(&trezor.Ping{PinProtection: &askPin, PassphraseProtection: &askPassphrase}, new(trezor.PinMatrixRequest), new(trezor.PassphraseRequest), new(trezor.Success))
	// 	if err != nil {
	// 		return err
	// 	}
	// 	// Only return the PIN request if the device wasn't unlocked until now
	// 	switch res {
	// 	case 0:
	// 		d.pinwait = true
	// 		return ErrTrezorPINNeeded
	// 	case 1:
	// 		d.pinwait = false
	// 		d.passphrasewait = true
	// 		return ErrTrezorPassphraseNeeded
	// 	case 2:
	// 		return nil // responded with trezor.Success
	// 	}
	// }
	// // Phase 2 requested with actual PIN entry
	// if d.pinwait {
	// 	d.pinwait = false
	// 	res, err := d.trezorExchange(&trezor.PinMatrixAck{Pin: &passphrase}, new(trezor.Success), new(trezor.PassphraseRequest))
	// 	if err != nil {
	// 		d.failure = err
	// 		return err
	// 	}
	// 	if res == 1 {
	// 		d.passphrasewait = true
	// 		return ErrTrezorPassphraseNeeded
	// 	}
	// } else if d.passphrasewait {
	// 	d.passphrasewait = false
	// 	if _, err := d.trezorExchange(&trezor.PassphraseAck{Passphrase: &passphrase}, new(trezor.Success)); err != nil {
	// 		d.failure = err
	// 		return err
	// 	}
	// }

	return nil
}
