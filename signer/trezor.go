package signer

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/address"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/core"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/signer/trezorproto"
	"github.com/ava-labs/coreth/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
)

type ConnectedDevice struct {
	*trezorproto.Features
}

type TrezorDriver struct {
	KnownDevices map[string]*ConnectedDevice
}

func NewTrezorDriver() TrezorDriver {
	return TrezorDriver{
		KnownDevices: make(map[string]*ConnectedDevice), // usb path -> dev
	}
}

func (d *TrezorDriver) Open(s *Signer) (core.USBDevice, error) {

	log.Trace().Msgf("Opening trezor: %s", s.Name)

	usb_path := ""

	for path, kd := range d.KnownDevices {
		if kd.Label != nil && *kd.Label == s.Name {
			usb_path = path
			break
		}
	}

	log.Trace().Msgf("Trezor known path: %s", usb_path)

	if usb_path == "" { // try one enumeration to find th edevice before asking the user
		l, err := cmn.Core.Enumerate()
		if err != nil {
			log.Error().Err(err).Msg("Error listing usb devices")
			return nil, err
		}

		for _, info := range l {
			if GetType(info.Vendor, info.Product) == "trezor" {
				n, err := GetDeviceName(info)
				if err != nil {
					log.Error().Err(err).Msg("Error getting device name")
					return nil, err
				}
				if cmn.IsInArray(s.GetFamilyNames(), n) {
					usb_path = info.Path
					log.Trace().Msgf("Trezor found on silent try: %s", s.Name)
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
			log.Debug().Msgf("Opened trezor: %s", s.Name)
			return dev, nil
		}
	}

	log.Trace().Msgf("Trezor not found: %s", s.Name)

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

	cmn.HailAndWait(&cmn.HailRequest{
		Title: "Connect Trezor",
		Template: `<c><w>
Please connect your Trezor device:

<u><b>` + s.Name + `</b></u>` + copies + `

<button text:Cancel>`,
		OnTick: func(h *cmn.HailRequest, tick int) {
			if tick%3 == 0 {
				log.Trace().Msg("Connect Trezor: Tick...")
				l, err := cmn.Core.Enumerate()
				if err != nil {
					log.Error().Err(err).Msg("Error listing usb devices")
					return
				}

				for _, info := range l {
					if GetType(info.Vendor, info.Product) == "trezor" {
						n, err := GetDeviceName(info)
						if err != nil {
							log.Error().Err(err).Msg("Error getting device name")
							return
						}
						if cmn.IsInArray(s.GetFamilyNames(), n) {
							usb_path = info.Path
							h.Close()
							return
						}
					}
				}
			}
		},
	})

	if usb_path == "" {
		log.Error().Msg("Open: Trezor not found")
		return nil, errors.New("trezor not found")
	}

	return cmn.Core.GetDevice(usb_path)
}

func (d TrezorDriver) IsConnected(s *Signer) bool {
	for _, kd := range d.KnownDevices {
		if kd.Label != nil && *kd.Label == s.Name {
			return true
		}
	}
	return false
}

func (d TrezorDriver) GetAddresses(s *Signer, path_format string, start_from int, count int) ([]address.Address, error) {
	r := []address.Address{}

	log.Trace().Msgf("Getting addresses for %s", s.Name)

	dev, err := d.Open(s)
	if err != nil {
		return r, err
	}

	log.Trace().Msgf("Device opened. Trying getting the addresses")

	for i := 0; i < count; i++ {
		path := fmt.Sprintf(path_format, start_from+i)
		dp, err := accounts.ParseDerivationPath(path)
		if err != nil {
			return r, err
		}

		eth_addr := new(trezorproto.EthereumAddress)
		if err := d.Call(dev,
			&trezorproto.EthereumGetAddress{AddressN: []uint32(dp)}, eth_addr); err != nil {
			return r, err
		}

		var a common.Address

		if addr := eth_addr.GetXOldAddress(); len(addr) > 0 { // Older firmwares use binary formats
			a = common.BytesToAddress(addr)
		}
		if addr := eth_addr.GetAddress(); len(addr) > 0 { // Newer firmwares use hexadecimal formats
			a = common.HexToAddress(addr)
		}

		r = append(r, address.Address{
			Address: a,
			Path:    path,
		})
	}
	return r, nil
}

// Type returns the protocol buffer type number of a specific message. If the
// message is nil, this method panics!
func MessageType(msg proto.Message) trezorproto.MessageType {

	return trezorproto.MessageType(trezorproto.MessageType_value["MessageType_"+reflect.TypeOf(msg).Elem().Name()])
}

// Name returns the friendly message type name of a specific protocol buffer
// type number.
func MessageName(kind trezorproto.MessageType) string {
	name := trezorproto.MessageType_name[int32(kind)]
	if len(name) < 12 {
		return name
	}
	return name[12:]
}

// errTrezorReplyInvalidHeader is the error message returned by a Trezor data exchange
// if the dev replies with a mismatching header. This usually means the dev
// is in browser mode.
var errTrezorReplyInvalidHeader = errors.New("trezor: invalid reply header")

func (d TrezorDriver) GetName(path string) (string, error) {

	kd, ok := d.KnownDevices[path]
	if ok {
		return *kd.Label, nil
	}

	cd, err := d.Init(path)
	if err != nil {
		return "", err
	} else {
		if cd.Label == nil {
			return "My Trezor", nil
		}

		return *cd.Label, nil
	}
}

func SS(s *string) string { // Safe string
	if s == nil {
		return ""
	}
	return *s
}

func SU32(s *uint32) uint32 { // Safe uint32
	if s == nil {
		return 0
	}
	return *s
}

func SB(s *bool) bool { // Safe bool
	if s == nil {
		return false
	}
	return *s
}

func (d TrezorDriver) Init(path string) (*ConnectedDevice, error) {
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
		if v.Label != nil && *v.Label == *features.Label {
			delete(d.KnownDevices, k)
		}
	}

	cd := ConnectedDevice{
		Features: features,
	}

	d.KnownDevices[path] = &cd
	//	log.Trace().Msgf("Initialized trezor dev: %v\n", *(cd.Label))
	return &cd, nil
}

func (d TrezorDriver) RequsetPin() (string, error) {
	template := "<c><w>\n<l id:pin text:'____________'> <button text:'\U000f006e ' id:back>\n\n"

	ids := []int{7, 8, 9, 4, 5, 6, 1, 2, 3}

	for i := 0; i < 9; i++ {
		template += fmt.Sprintf("<button color:g.HelpFgColor bgcolor:g.HelpBgColor text:' - ' id:%d> ", ids[i])
		if (i+1)%3 == 0 {
			template += "\n\n"
		}
	}
	template += "<button text:OK> <button text:Cancel>"
	pin := ""

	cmn.HailAndWait(&cmn.HailRequest{
		Title:    "Enter Trezor PIN",
		Template: template,
		OnClickHotspot: func(h *cmn.HailRequest, v *gocui.View, hs *gocui.Hotspot) {
			if hs != nil {
				s := cmn.Split(hs.Value)
				command, value := s[0], s[1]

				switch command {
				case "button":
					switch value {
					case "back":
						if len(pin) > 0 {
							pin = pin[:len(pin)-1]
							v.GetHotspotById("pin").SetText(strings.Repeat("*", len(pin)) + "______________")
						}
					case "1", "2", "3", "4", "5", "6", "7", "8", "9":
						pin += value
						v.GetHotspotById("pin").SetText(strings.Repeat("*", len(pin)) + "______________")
					}
				}
			}
		},
	})

	if pin == "" {
		return "", errors.New("pin request canceled")
	}

	return pin, nil

}

func (d TrezorDriver) RequsetPassword() (string, error) {
	password := ""
	canceled := false

	cmn.HailAndWait(&cmn.HailRequest{
		Title: "Select Wallet Type",
		Template: `<c><w>
<button text:Standard color:g.HelpFgColor bgcolor:g.HelpBgColor id:standard> <button text:Hidden color:g.HelpFgColor bgcolor:g.HelpBgColor id:hidden> 

<button text:Cancel>`,

		OnClickHotspot: func(h *cmn.HailRequest, v *gocui.View, hs *gocui.Hotspot) {
			if hs != nil {
				s := cmn.Split(hs.Value)
				command, value := s[0], s[1]

				switch command {
				case "button":
					switch value {
					case "standard":
						h.Close()
					case "hidden":
						h.TimerPaused = true
						v.GetGui().ShowPopup(&gocui.Popup{
							Title: "Enter Trezor Password",
							Template: `<c><w>
Password: <i id:password size:16 masked:true>

<button text:OK> <button text:Cancel>`,
							OnClickHotspot: func(v *gocui.View, hs *gocui.Hotspot) {
								if hs != nil {
									switch hs.Value {
									case "button OK":
										password = v.GetInput("password")
										v.GetGui().HidePopup()
										h.Close()
									case "button Cancel":
										v.GetGui().HidePopup()
									}
								}
							},
						})
						h.TimerPaused = false
					}
				}
			}
		},
		OnCancel: func(h *cmn.HailRequest) {
			canceled = true
		},
	})

	if canceled {
		return "", errors.New("password request canceled")
	}

	log.Debug().Msgf("Password: %s", password)

	return password, nil
}

func (d TrezorDriver) Call(dev core.USBDevice, req proto.Message, result proto.Message) error {
	log.Debug().Msgf("Call: %s", MessageName(MessageType(req)))
	log.Debug().Msgf("Call: %v", req)

	kind, reply, err := d.RawCall(dev, req)
	if err != nil {
		log.Error().Msgf("Call: Error calling device: %s", err)
		return err
	}
	for {
		switch kind {
		case trezorproto.MessageType_MessageType_PinMatrixRequest:
			{
				log.Trace().Msg("*** NB! Enter PIN (not echoed)...")
				pin, err := d.RequsetPin()
				if err != nil {
					log.Error().Msgf("Call: Error getting pin: %s", err)
					d.RawCall(dev, &trezorproto.Cancel{})
					return err
				}

				pinStr := string(pin)
				for _, ch := range pinStr {
					if !strings.ContainsRune("123456789", ch) || len(pin) < 1 {
						log.Error().Msgf("Call: Invalid PIN provided")
						d.RawCall(dev, &trezorproto.Cancel{})
						return errors.New("trezor: Invalid PIN provided")
					}
				}
				// send pin
				kind, reply, err = d.RawCall(dev, &trezorproto.PinMatrixAck{Pin: &pinStr})
				if err != nil {
					log.Error().Msgf("Call: Error sending pin: %s", err)
					return err
				}
				log.Trace().Msgf("Trezor pin success. kind: %s\n", MessageName(kind))
			}
		case trezorproto.MessageType_MessageType_PassphraseRequest:
			{
				log.Trace().Msg("*** NB! Enter Pass	phrase ...")
				pass, err := d.RequsetPassword()
				if err != nil {
					d.RawCall(dev, &trezorproto.Cancel{})
					return err
				}
				passStr := string(pass)
				// send it
				kind, reply, err = d.RawCall(dev, &trezorproto.PassphraseAck{Passphrase: &passStr})
				if err != nil {
					return err
				}
				log.Trace().Msgf("Trezor pass success. kind: %s\n", MessageName(kind))
			}
		case trezorproto.MessageType_MessageType_ButtonRequest:
			{
				log.Trace().Msg("*** NB! Button request on your Trezor screen ...")
				// Trezor is waiting for user confirmation, ack and wait for the next message
				kind, reply, err = d.RawCall(dev, &trezorproto.ButtonAck{})
				if err != nil {
					return err
				}
				log.Trace().Msgf("Trezor button success. kind: %s\n", MessageName(kind))
			}
		case trezorproto.MessageType_MessageType_Failure:
			{
				// Trezor returned a failure, extract and return the message
				failure := new(trezorproto.Failure)
				if err := proto.Unmarshal(reply, failure); err != nil {
					return err
				}
				// fmt.Printf("Trezor failure success. kind: %s\n", MessageName(kind))
				return errors.New("trezor: " + failure.GetMessage())
			}
		default:
			{
				resultKind := MessageType(result)
				if resultKind != kind {
					return fmt.Errorf("trezor: expected reply type %s, got %s", MessageName(resultKind), MessageName(kind))
				}
				return proto.Unmarshal(reply, result)
			}
		}
	}
}

func (d TrezorDriver) RawCall(dev core.USBDevice, req proto.Message) (trezorproto.MessageType, []byte, error) {

	log.Debug().Msgf("RawCall: %s", MessageName(MessageType(req)))

	data, err := proto.Marshal(req)
	if err != nil {
		log.Error().Msgf("RawCall: Error marshalling request: %s", err)
		return 0, nil, err
	}

	payload := make([]byte, 8+len(data))
	copy(payload, []byte{0x23, 0x23})
	binary.BigEndian.PutUint16(payload[2:], uint16(MessageType(req)))
	binary.BigEndian.PutUint32(payload[4:], uint32(len(data)))
	copy(payload[8:], data)

	// Stream all the chunks to the dev
	chunk := make([]byte, 64)
	chunk[0] = 0x3f // Report ID magic number

	for len(payload) > 0 {
		// Construct the new message to stream, padding with zeroes if needed
		if len(payload) > 63 {
			copy(chunk[1:], payload[:63])
			payload = payload[63:]
		} else {
			copy(chunk[1:], payload)
			copy(chunk[1+len(payload):], make([]byte, 63-len(payload)))
			payload = nil
		}
		// Send over to the dev
		// log.Trace().Msgf("Data chunk sent to the Trezor: %v\n", hexutil.Bytes(chunk))
		if _, err := dev.Write(chunk); err != nil {
			log.Error().Msgf("RawCall: Error writing to device: %s", err)
			return 0, nil, err
		}
	}

	// Stream the reply back from the wallet in 64 byte chunks
	var (
		kind  uint16
		reply []byte
	)
	for {

		// Read the next chunk from the Trezor wallet
		if _, err := io.ReadFull(dev, chunk); err != nil {
			log.Error().Msgf("RawCall: Error reading from device: %s", err)
			return 0, nil, err
		}

		// Make sure the transport header matches
		if chunk[0] != 0x3f || (len(reply) == 0 && (chunk[1] != 0x23 || chunk[2] != 0x23)) {
			log.Error().Msgf("RawCall: Invalid reply header: %v", chunk)
			return 0, nil, errTrezorReplyInvalidHeader
		}
		// If it's the first chunk, retrieve the reply message type and total message length
		var payload []byte

		if len(reply) == 0 {
			kind = binary.BigEndian.Uint16(chunk[3:5])
			reply = make([]byte, 0, int(binary.BigEndian.Uint32(chunk[5:9])))
			payload = chunk[9:]
		} else {
			payload = chunk[1:]
		}
		// Append to the reply and stop when filled up
		if left := cap(reply) - len(reply); left > len(payload) {
			reply = append(reply, payload...)
		} else {
			reply = append(reply, payload[:left]...)
			break
		}

		log.Trace().Msg("3")

	}

	return trezorproto.MessageType(kind), reply, nil
}

func (d TrezorDriver) PrintDetails(path string) string {
	dev, ok := d.KnownDevices[path]
	if !ok {
		log.Error().Msgf("PrintDetails: Device not found: %s", path)
		return ""
	}

	r := ""
	r += fmt.Sprintf("  Vendor: %s\n", SS(dev.Vendor))
	r += fmt.Sprintf("  MajorVersion: %d\n", SU32(dev.MajorVersion))
	r += fmt.Sprintf("  MinorVersion: %d\n", SU32(dev.MinorVersion))
	r += fmt.Sprintf("  PatchVersion: %d\n", SU32(dev.PatchVersion))
	// r += fmt.Sprintf("  BootloaderMode: %t\n", SB(dev.BootloaderMode))
	r += fmt.Sprintf("  DeviceId: %s\n", SS(dev.DeviceId))
	r += fmt.Sprintf("  PinProtection: %t\n", SB(dev.PinProtection))
	r += fmt.Sprintf("  PassphraseProtection: %t\n", SB(dev.PassphraseProtection))
	r += fmt.Sprintf("  Language: %s\n", SS(dev.Language))
	r += fmt.Sprintf("  Label: %s\n", SS(dev.Label))
	r += fmt.Sprintf("  Initialized: %t\n", SB(dev.Initialized))
	// r += fmt.Sprintf("  Revision: %v\n", dev.Revision)
	// r += fmt.Sprintf("  BootloaderHash: %v\n", dev.BootloaderHash)
	r += fmt.Sprintf("  Imported: %t\n", SB(dev.Imported))
	r += fmt.Sprintf("  FirmwarePresent: %t\n", SB(dev.FirmwarePresent))
	r += fmt.Sprintf("  NeedsBackup: %t\n", SB(dev.NeedsBackup))
	r += fmt.Sprintf("  Flags: %d\n", SU32(dev.Flags))
	r += fmt.Sprintf("  Model: %s\n", SS(dev.Model))
	// r += fmt.Sprintf("  FwMajor: %d\n", SU32(dev.FwMajor))
	// r += fmt.Sprintf("  FwMinor: %d\n", SU32(dev.FwMinor))
	// r += fmt.Sprintf("  FwPatch: %d\n", SU32(dev.FwPatch))
	r += fmt.Sprintf("  FwVendor: %s\n", SS(dev.FwVendor))
	r += fmt.Sprintf("  FwVendorKeys: %v\n", dev.FwVendorKeys)
	// r += fmt.Sprintf("  UnfinishedBackup: %t\n", SB(dev.UnfinishedBackup))
	// r += fmt.Sprintf("  NoBackup: %t\n", SB(dev.NoBackup))

	return r
}
