package trezor

import (
	"encoding/binary"
	"errors"
	"reflect"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/hw/trezor/trezorproto"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
)

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

func (d Trezor) RawCall(msg *bus.Message, req proto.Message) (trezorproto.MessageType, []byte, error) {
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
		log.Trace().Msgf("Data chunk sent to the Trezor: %v\n", hexutil.Bytes(chunk))
		resp := msg.Fetch("usb", "write", &bus.B_UsbWrite{
			USB_ID: d.USB_ID,
			Data:   chunk,
		})
		if resp.Error != nil {
			log.Error().Msgf("RawCall: Error writing to device: %s", resp.Error)
			return 0, nil, resp.Error
		}
	}

	// Stream the reply back from the wallet in 64 byte chunks
	var (
		kind  uint16
		reply []byte
	)
	for {
		resp := msg.Fetch("usb", "read", &bus.B_UsbRead{
			USB_ID: d.USB_ID,
		})
		if resp.Error != nil {
			log.Error().Msgf("RawCall: Error reading from device: %s", resp.Error)
			return 0, nil, resp.Error
		}
		chunk := resp.Data.(*bus.B_UsbRead_Response).Data

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
	}

	return trezorproto.MessageType(kind), reply, nil
}
