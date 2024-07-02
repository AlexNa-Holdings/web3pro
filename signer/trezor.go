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
	"github.com/ava-labs/coreth/accounts"
	"github.com/ethereum/go-ethereum/accounts/usbwallet/trezor"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
)

type ConnectedDevice struct {
	trezor.Features
}

type TrezorDriver struct {
	KnownDevices map[string]ConnectedDevice
}

func NewTrezorDriver() TrezorDriver {
	return TrezorDriver{
		KnownDevices: make(map[string]ConnectedDevice), // usb path -> dev
	}
}

func (d *TrezorDriver) Open(s *Signer) (core.USBDevice, error) {

	log.Trace().Msgf("Opening trezor: %s\n", s.Name)

	usb_path := ""

	for path, kd := range d.KnownDevices {
		if kd.Label != nil && *kd.Label == s.Name {
			usb_path = path
			break
		}
	}

	log.Trace().Msgf("Trezor known path: %s\n", usb_path)

	if usb_path != "" {
		dev, err := cmn.Core.GetDevice(usb_path)
		if err != nil {
			// probably disconnected
			log.Error().Err(err).Msgf("Error (ignored) getting device: %v", err)
			delete(d.KnownDevices, usb_path)
			usb_path = ""
		} else {
			log.Debug().Msgf("Opened trezor: %s\n", s.Name)
			return dev, nil
		}
	}

	log.Trace().Msgf("Trezor not found: %s\n", s.Name)

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
		return nil, errors.New("trezor not found")
	}

	log.Debug().Msgf("Opening trezor: %s\n", usb_path)
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

	log.Trace().Msgf("Getting addresses for %s\n", s.Name)

	dev, err := d.Open(s)
	if err != nil {
		return r, err
	}

	log.Trace().Msgf("Opened trezor: %s", s.Name)

	for i := 0; i < count; i++ {
		path := fmt.Sprintf(path_format, start_from+i)
		dp, err := accounts.ParseDerivationPath(path)
		if err != nil {
			return r, err
		}

		eth_addr := V2Of(new(trezor.EthereumAddress))
		if err := d.Call(dev,
			V2Of(&trezor.EthereumGetAddress{AddressN: []uint32(dp)}), eth_addr); err != nil {
			return r, err
		}

		ethAddress := V1Of(eth_addr).(*trezor.EthereumAddress)

		var a common.Address
		if len(ethAddress.AddressBin) > 0 { // Older firmwares use binary formats
			a = common.BytesToAddress(ethAddress.AddressBin)
		} else {
			if ethAddress.AddressHex == nil {
				return r, errors.New("trezor: nil address returned")
			}

			a = common.HexToAddress(*ethAddress.AddressHex)
		}

		r = append(r, address.Address{Address: a})
	}
	return r, nil
}

// Type returns the protocol buffer type number of a specific message. If the
// message is nil, this method panics!
func MessageType(msg proto.Message) trezor.MessageType {
	return trezor.MessageType(trezor.MessageType_value["MessageType_"+reflect.TypeOf(msg).Elem().Name()])
}

// Name returns the friendly message type name of a specific protocol buffer
// type number.
func MessageName(kind trezor.MessageType) string {
	name := trezor.MessageType_name[int32(kind)]
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

	kind, reply, err := d.RawCall(dev, V2Of(&trezor.Initialize{}))
	if err != nil {
		log.Error().Err(err).Msgf("Init: Error initializing device: %s", path)
		return nil, err
	}
	if kind != trezor.MessageType_MessageType_Features {
		log.Error().Msgf("Init: Expected reply type %s, got %s", MessageName(trezor.MessageType_MessageType_Features), MessageName(kind))
		return nil, errors.New("trezor: expected reply type " + MessageName(trezor.MessageType_MessageType_Features) + ", got " + MessageName(kind))
	}
	features := new(trezor.Features)
	err = proto.Unmarshal(reply, V2Of(features))
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
		Features: *features,
	}

	d.KnownDevices[path] = cd
	//	log.Trace().Msgf("Initialized trezor dev: %v\n", *(cd.Label))
	return &cd, nil
}

func (d TrezorDriver) RequsetPin() (string, error) {
	cmn.HailAndWait(&cmn.HailRequest{
		Title: "Enter Trezor PIN",
		Template: `<c><w>

<button text="###"> <button text="###"> <button text="###"> 
<button text="###"> <button text="###"> <button text="###"> 
<button text="###"> <button text="###"> <button text="###"> 

		
<button text="OK"> <button text="Cancel">
		`,
	})

	return "", nil

}

func (d TrezorDriver) Call(dev core.USBDevice, req proto.Message, result proto.Message) error {

	log.Debug().Msgf("Trezor call: %s", MessageName(MessageType(req)))

	kind, reply, err := d.RawCall(dev, req)
	if err != nil {
		return err
	}
	for {
		// fmt.Printf("for loop new call. kind: %s ...\n", MessageName(kind))
		switch kind {
		case trezor.MessageType_MessageType_PinMatrixRequest:
			{
				log.Trace().Msg("*** NB! Enter PIN (not echoed)...")
				pin, _ := d.RequsetPin()
				// pin, err := w.ui.ReadPassword()
				// if err != nil {
				// 	kind, reply, _ = w.rawCall(&trezor.Cancel{})
				// 	return err
				// }
				// check if pin is valid
				pinStr := string(pin)
				for _, ch := range pinStr {
					if !strings.ContainsRune("123456789", ch) || len(pin) < 1 {
						d.RawCall(dev, V2Of(&trezor.Cancel{}))
						return errors.New("trezor: Invalid PIN provided")
					}
				}
				// send pin
				kind, reply, err = d.RawCall(dev, V2Of(&trezor.PinMatrixAck{Pin: &pinStr}))
				if err != nil {
					return err
				}
				log.Trace().Msgf("Trezor pin success. kind: %s\n", MessageName(kind))
			}
		case trezor.MessageType_MessageType_PassphraseRequest:
			{
				log.Trace().Msg("*** NB! Enter Pass	phrase ...")
				pass := "12345" // TODOphrase ...")
				// pass, err := w.ui.ReadPassword()
				// if err != nil {
				// 	kind, reply, _ = d.RawCall(dev, V2Of(&trezor.Cancel{}))
				// 	return err
				// }
				passStr := string(pass)
				// send it
				kind, reply, err = d.RawCall(dev, V2Of(&trezor.PassphraseAck{Passphrase: &passStr}))
				if err != nil {
					return err
				}
				log.Trace().Msgf("Trezor pass success. kind: %s\n", MessageName(kind))
			}
		case trezor.MessageType_MessageType_ButtonRequest:
			{
				log.Trace().Msg("*** NB! Button request on your Trezor screen ...")
				// Trezor is waiting for user confirmation, ack and wait for the next message
				kind, reply, err = d.RawCall(dev, V2Of(&trezor.ButtonAck{}))
				if err != nil {
					return err
				}
				log.Trace().Msgf("Trezor button success. kind: %s\n", MessageName(kind))
			}
		case trezor.MessageType_MessageType_Failure:
			{
				// Trezor returned a failure, extract and return the message
				failure := new(trezor.Failure)
				if err := proto.Unmarshal(reply, V2Of(failure)); err != nil {
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

func (d TrezorDriver) RawCall(dev core.USBDevice, req proto.Message) (trezor.MessageType, []byte, error) {
	data, err := proto.Marshal(req)
	if err != nil {
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
			return 0, nil, err
		}

		// Make sure the transport header matches
		if chunk[0] != 0x3f || (len(reply) == 0 && (chunk[1] != 0x23 || chunk[2] != 0x23)) {
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
	}
	return trezor.MessageType(kind), reply, nil
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
	r += fmt.Sprintf("  PinCached: %t\n", SB(dev.PinCached))
	r += fmt.Sprintf("  PassphraseCached: %t\n", SB(dev.PassphraseCached))
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
