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
	"github.com/ethereum/go-ethereum/accounts/usbwallet/trezor"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/protoadapt"
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

func (d *TrezorDriver) Open() error {

	return nil
}

func (d TrezorDriver) IsConnected(s *Signer) bool {
	//TODO
	return false
}

func (d TrezorDriver) GetAddresses(s *Signer, path_format string, start_from int, count int) ([]address.Address, error) {
	return []address.Address{}, nil // TODO
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
		return *cd.Label, nil
	}
}

func (d TrezorDriver) Init(path string) (*ConnectedDevice, error) {
	dev, err := cmn.Core.GetDevice(path)
	if err != nil {
		log.Error().Err(err).Msgf("Init: Error getting device: %s", path)
		return nil, err
	}

	kind, reply, err := d.RawCall(dev, protoadapt.MessageV2Of(&trezor.Initialize{}))
	if err != nil {
		log.Error().Err(err).Msgf("Init: Error initializing device: %s", path)
		return nil, err
	}
	if kind != trezor.MessageType_MessageType_Features {
		log.Error().Msgf("Init: Expected reply type %s, got %s", MessageName(trezor.MessageType_MessageType_Features), MessageName(kind))
		return nil, errors.New("trezor: expected reply type " + MessageName(trezor.MessageType_MessageType_Features) + ", got " + MessageName(kind))
	}
	features := new(trezor.Features)
	err = proto.Unmarshal(reply, protoadapt.MessageV2Of(features))
	if err != nil {
		log.Error().Err(err).Msgf("Init: Error unmarshalling features: %s", path)
		return nil, err
	}

	// remove from tge KnownDevices all with the same lab label
	for k, v := range d.KnownDevices {
		if *v.Label == *features.Label {
			delete(d.KnownDevices, k)
		}
	}

	cd := ConnectedDevice{
		Features: *features,
	}

	d.KnownDevices[path] = cd
	log.Trace().Msgf("Initialized trezor dev: %s\n", *cd.Label)
	return &cd, nil
}

func (d TrezorDriver) Call(dev core.USBDevice, req proto.Message, result proto.Message) error {
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
				pin := "123456" //TODO
				// pin, err := w.ui.ReadPassword()
				// if err != nil {
				// 	kind, reply, _ = w.rawCall(&trezor.Cancel{})
				// 	return err
				// }
				// check if pin is valid
				pinStr := string(pin)
				for _, ch := range pinStr {
					if !strings.ContainsRune("123456789", ch) || len(pin) < 1 {
						kind, reply, _ = d.RawCall(dev, protoadapt.MessageV2Of(&trezor.Cancel{}))
						return errors.New("trezor: Invalid PIN provided")
					}
				}
				// send pin
				kind, reply, err = d.RawCall(dev, protoadapt.MessageV2Of(&trezor.PinMatrixAck{Pin: &pinStr}))
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
				// 	kind, reply, _ = d.RawCall(dev, protoadapt.MessageV2Of(&trezor.Cancel{}))
				// 	return err
				// }
				passStr := string(pass)
				// send it
				kind, reply, err = d.RawCall(dev, protoadapt.MessageV2Of(&trezor.PassphraseAck{Passphrase: &passStr}))
				if err != nil {
					return err
				}
				log.Trace().Msgf("Trezor pass success. kind: %s\n", MessageName(kind))
			}
		case trezor.MessageType_MessageType_ButtonRequest:
			{
				log.Trace().Msg("*** NB! Button request on your Trezor screen ...")
				// Trezor is waiting for user confirmation, ack and wait for the next message
				kind, reply, err = d.RawCall(dev, protoadapt.MessageV2Of(&trezor.ButtonAck{}))
				if err != nil {
					return err
				}
				log.Trace().Msgf("Trezor button success. kind: %s\n", MessageName(kind))
			}
		case trezor.MessageType_MessageType_Failure:
			{
				// Trezor returned a failure, extract and return the message
				failure := new(trezor.Failure)
				if err := proto.Unmarshal(reply, protoadapt.MessageV2Of(failure)); err != nil {
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

// func (w *TrezorDriver) Call(req proto.Message, result proto.Message) error {
// 	kind, reply, err := w.rawCall(req)
// 	if err != nil {
// 		return err
// 	}
// 	for {
// 		// fmt.Printf("for loop new call. kind: %s ...\n", MessageName(kind))
// 		switch kind {
// 		case trezor.MessageType_MessageType_PinMatrixRequest:
// 			{
// 				log.Trace().Msg("*** NB! Enter PIN (not echoed)...")
// 				log.Trace().Msg(PIN_MATRIX)
// 				pin, err := w.ui.ReadPassword()
// 				if err != nil {
// 					kind, reply, _ = w.rawCall(&trezor.Cancel{})
// 					return err
// 				}
// 				// check if pin is valid
// 				pinStr := string(pin)
// 				for _, d := range pinStr {
// 					if !strings.ContainsRune("123456789", d) || len(pin) < 1 {
// 						kind, reply, _ = w.rawCall(&trezor.Cancel{})
// 						return errors.New("trezor: Invalid PIN provided")
// 					}
// 				}
// 				// send pin
// 				kind, reply, err = w.rawCall(&trezor.PinMatrixAck{Pin: &pinStr})
// 				if err != nil {
// 					return err
// 				}
// 				log.Trace().Msgf("Trezor pin success. kind: %s\n", MessageName(kind))
// 			}
// 		case trezor.MessageType_MessageType_PassphraseRequest:
// 			{
// 				log.Trace().Msg("*** NB! Enter Passphrase ...")
// 				pass, err := w.ui.ReadPassword()
// 				if err != nil {
// 					kind, reply, _ = w.rawCall(&trezor.Cancel{})
// 					return err
// 				}
// 				passStr := string(pass)
// 				// send it
// 				kind, reply, err = w.rawCall(&trezor.PassphraseAck{Passphrase: &passStr})
// 				if err != nil {
// 					return err
// 				}
// 				log.Trace().Msgf("Trezor pass success. kind: %s\n", MessageName(kind))
// 			}
// 		case trezor.MessageType_MessageType_ButtonRequest:
// 			{
// 				log.Trace().Msg("*** NB! Button request on your Trezor screen ...")
// 				// Trezor is waiting for user confirmation, ack and wait for the next message
// 				kind, reply, err = w.rawCall(&trezor.ButtonAck{})
// 				if err != nil {
// 					return err
// 				}
// 				log.Trace().Msgf("Trezor button success. kind: %s\n", MessageName(kind))
// 			}
// 		case trezor.MessageType_MessageType_Failure:
// 			{
// 				// Trezor returned a failure, extract and return the message
// 				failure := new(trezor.Failure)
// 				if err := proto.Unmarshal(reply, failure); err != nil {
// 					return err
// 				}
// 				// fmt.Printf("Trezor failure success. kind: %s\n", MessageName(kind))
// 				return errors.New("trezor: " + failure.GetMessage())
// 			}
// 		default:
// 			{
// 				resultKind := MessageType(result)
// 				if resultKind != kind {
// 					return fmt.Errorf("trezor: expected reply type %s, got %s", MessageName(resultKind), MessageName(kind))
// 				}
// 				return proto.Unmarshal(reply, result)
// 			}
// 		}
// 	}
// }
