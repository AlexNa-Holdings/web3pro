package signer_driver

import (
	"errors"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/signer_driver/trezorproto"
	"github.com/AlexNa-Holdings/web3pro/usb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
)

type ConnectedLedger struct {
	Label string
}

type LedgerDriver struct {
	KnownDevices map[string]*ConnectedLedger
}

func NewLedgerDriver() LedgerDriver {
	return LedgerDriver{
		KnownDevices: make(map[string]*ConnectedLedger), // usb path -> dev
	}
}

func (d *LedgerDriver) Open(s *cmn.Signer) (usb.USBDevice, error) {

	log.Trace().Msgf("Opening ledger: %s", s.Name)

	usb_path := ""

	for path, kd := range d.KnownDevices {
		if kd.Label == s.Name {
			usb_path = path
			break
		}
	}

	log.Trace().Msgf("Ledger known path: %s", usb_path)

	if usb_path == "" { // try one enumeration to find th edevice before asking the user
		l, err := cmn.Core.Enumerate()
		if err != nil {
			log.Error().Err(err).Msg("Error listing usb devices")
			return nil, err
		}

		for _, info := range l {
			if cmn.GetDeviceType(info.Vendor, info.Product) == "ledger" {
				n, err := cmn.GetDeviceName(info)
				if err != nil {
					log.Error().Err(err).Msg("Error getting device name")
					return nil, err
				}
				if cmn.IsInArray(s.GetFamilyNames(), n) {
					usb_path = info.Path
					log.Trace().Msgf("Ledger found on silent try: %s", s.Name)
					break
				}
			}
		}
	}

	if usb_path != "" {
		dev, err := cmn.Core.GetDevice(usb_path)
		if err != nil {
			// probably disconnected
			log.Error().Err(err).Msgf("Error (ignored) getting device: %v", err)
			delete(d.KnownDevices, usb_path)
			usb_path = ""
		} else {
			return dev, nil
		}
	}

	log.Trace().Msgf("Ledger not found: %s", s.Name)

	copies := ""
	if len(s.Copies) > 0 {
		copies = "\n or one of the copies:\n<u><b>"
		for i, c := range s.Copies {
			copies += c
			if i < len(s.Copies)-1 {
				copies += ", "
			}
		}
		copies += "</b></u>"
	}

	bus.Fetch("ui", "hail", &bus.B_Hail{
		Title: "Connect Ledger",
		Template: `<c><w>
Please connect your Ledger device:

<u><b>` + s.Name + `</b></u>` + copies + `

<button text:Cancel>`,
		OnTick: func(h *bus.B_Hail, tick int) {
			if tick%3 == 0 {
				log.Trace().Msg("Connect Ledger: Tick...")
				l, err := cmn.Core.Enumerate()
				if err != nil {
					log.Error().Err(err).Msg("Error listing usb devices")
					return
				}

				for _, info := range l {
					if cmn.GetDeviceType(info.Vendor, info.Product) == "ledger" {
						n, err := cmn.GetDeviceName(info)
						if err != nil {
							log.Error().Err(err).Msg("Error getting device name")
							return
						}
						if cmn.IsInArray(s.GetFamilyNames(), n) {
							usb_path = info.Path
							bus.Send("ui", "remove_hail", h)
							return
						}
					}
				}
			}
		},
	})

	if usb_path == "" {
		log.Error().Msg("Open: Ledger not found")
		return nil, errors.New("ledger not found")
	}

	return cmn.Core.GetDevice(usb_path)
}

func (d LedgerDriver) IsConnected(s *cmn.Signer) bool {
	for _, kd := range d.KnownDevices {
		if kd.Label == s.Name {
			return true
		}
	}
	return false
}

func (d LedgerDriver) GetAddresses(s *cmn.Signer, path_format string, start_from int, count int) ([]cmn.Address, error) {
	r := []cmn.Address{}

	log.Trace().Msgf("Getting addresses for %s", s.Name)

	dev, err := d.Open(s)
	if err != nil {
		return r, err
	}

	log.Trace().Msgf("Device opened. Trying getting the addresses")

	dev = dev
	// for i := 0; i < count; i++ {
	// 	path := fmt.Sprintf(path_format, start_from+i)
	// 	dp, err := accounts.ParseDerivationPath(path)
	// 	if err != nil {
	// 		return r, err
	// 	}

	// 	eth_addr := new(trezorproto.EthereumAddress)
	// 	if err := d.Call(dev,
	// 		&trezorproto.EthereumGetAddress{AddressN: []uint32(dp)}, eth_addr); err != nil {
	// 		return r, err
	// 	}

	// 	var a common.Address

	// 	if addr := eth_addr.GetXOldAddress(); len(addr) > 0 { // Older firmwares use binary formats
	// 		a = common.BytesToAddress(addr)
	// 	}
	// 	if addr := eth_addr.GetAddress(); len(addr) > 0 { // Newer firmwares use hexadecimal formats
	// 		a = common.HexToAddress(addr)
	// 	}

	// 	r = append(r, cmn.Address{
	// 		Address: a,
	// 		Path:    path,
	// 	})
	// }
	return r, nil
}

func (d LedgerDriver) GetName(path string) (string, error) {

	kd, ok := d.KnownDevices[path]
	if ok {
		return kd.Label, nil
	}

	cd, err := d.Init(path)
	if err != nil {
		return "", err
	} else {
		if cd.Label == "" {
			return "My Trezor", nil
		}
		return cd.Label, nil
	}
}

func (d LedgerDriver) Init(path string) (*ConnectedLedger, error) {
	dev, err := cmn.Core.GetDevice(path)
	if err != nil {
		log.Error().Err(err).Msgf("Init: Error getting device: %s", path)
		return nil, err
	}

	kind, reply, err := d.RawCall(dev, &trezorproto.Initialize{})
	if err != nil {
		log.Error().Err(err).Msgf("Init: Error initializing device: %s", path)
		return nil, err
	}
	if kind != trezorproto.MessageType_MessageType_Features {
		log.Error().Msgf("Init: Expected reply type %s, got %s", MessageName(trezorproto.MessageType_MessageType_Features), MessageName(kind))
		return nil, errors.New("trezor: expected reply type " + MessageName(trezorproto.MessageType_MessageType_Features) + ", got " + MessageName(kind))
	}
	features := new(trezorproto.Features)
	err = proto.Unmarshal(reply, features)
	if err != nil {
		log.Error().Err(err).Msgf("Init: Error unmarshalling features: %s", path)
		return nil, err
	}

	// remove from tge KnownDevices all with the same lab label
	for k, v := range d.KnownDevices {
		if v.Label == *features.Label {
			delete(d.KnownDevices, k)
		}
	}

	cd := ConnectedLedger{
		//Features: features,
	}

	d.KnownDevices[path] = &cd
	//	log.Trace().Msgf("Initialized trezor dev: %v\n", *(cd.Label))
	return &cd, nil
}

func (d LedgerDriver) Call(dev usb.USBDevice, req proto.Message, result proto.Message) error {
	return nil // Not implemented
}

func (d LedgerDriver) RawCall(dev usb.USBDevice, req proto.Message) (trezorproto.MessageType, []byte, error) {
	return 0, nil, nil // Not implemented
}

func (d LedgerDriver) PrintDetails(path string) string {
	_, ok := d.KnownDevices[path]
	if !ok {
		log.Error().Msgf("PrintDetails: Device not found: %s", path)
		return ""
	}

	r := ""
	// r += fmt.Sprintf("  Vendor: %s\n", SS(dev.Vendor))
	// r += fmt.Sprintf("  MajorVersion: %d\n", SU32(dev.MajorVersion))
	// r += fmt.Sprintf("  MinorVersion: %d\n", SU32(dev.MinorVersion))
	// r += fmt.Sprintf("  PatchVersion: %d\n", SU32(dev.PatchVersion))
	// r += fmt.Sprintf("  BootloaderMode: %t\n", SB(dev.BootloaderMode))
	// r += fmt.Sprintf("  DeviceId: %s\n", SS(dev.DeviceId))
	// r += fmt.Sprintf("  PinProtection: %t\n", SB(dev.PinProtection))
	// r += fmt.Sprintf("  PassphraseProtection: %t\n", SB(dev.PassphraseProtection))
	// r += fmt.Sprintf("  Language: %s\n", SS(dev.Language))
	// r += fmt.Sprintf("  Label: %s\n", SS(dev.Label))
	// r += fmt.Sprintf("  Initialized: %t\n", SB(dev.Initialized))
	// r += fmt.Sprintf("  Revision: %v\n", dev.Revision)
	// r += fmt.Sprintf("  BootloaderHash: %v\n", dev.BootloaderHash)
	// r += fmt.Sprintf("  Imported: %t\n", SB(dev.Imported))
	// r += fmt.Sprintf("  FirmwarePresent: %t\n", SB(dev.FirmwarePresent))
	// r += fmt.Sprintf("  NeedsBackup: %t\n", SB(dev.NeedsBackup))
	// r += fmt.Sprintf("  Flags: %d\n", SU32(dev.Flags))
	// r += fmt.Sprintf("  Model: %s\n", SS(dev.Model))
	// r += fmt.Sprintf("  FwMajor: %d\n", SU32(dev.FwMajor))
	// r += fmt.Sprintf("  FwMinor: %d\n", SU32(dev.FwMinor))
	// r += fmt.Sprintf("  FwPatch: %d\n", SU32(dev.FwPatch))
	// r += fmt.Sprintf("  FwVendor: %s\n", SS(dev.FwVendor))
	// r += fmt.Sprintf("  FwVendorKeys: %v\n", dev.FwVendorKeys)
	// r += fmt.Sprintf("  UnfinishedBackup: %t\n", SB(dev.UnfinishedBackup))
	// r += fmt.Sprintf("  NoBackup: %t\n", SB(dev.NoBackup))

	return r
}
func (d LedgerDriver) SignTx(b *cmn.Blockchain, s *cmn.Signer, tx *types.Transaction, a *cmn.Address) (*types.Transaction, error) {
	return nil, nil // Not implemented
}
