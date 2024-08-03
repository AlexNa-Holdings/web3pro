package ledger

import (
	"encoding/binary"
	"errors"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/rs/zerolog/log"
)

func rawCall(usb_id string, adpu *APDU) ([]byte, error) {
	opcode := adpu.cla
	p1 := adpu.ins
	p2 := adpu.p1
	data := adpu.data

	// Construct the message payload, possibly split into multiple chunks
	apdu := make([]byte, 2, 7+len(data))

	binary.BigEndian.PutUint16(apdu, uint16(5+len(data)))
	apdu = append(apdu, []byte{0xe0, byte(opcode), byte(p1), byte(p2), byte(len(data))}...)
	apdu = append(apdu, data...)

	// Stream all the chunks to the device
	header := []byte{0x01, 0x01, 0x05, 0x00, 0x00} // Channel ID and command tag appended
	chunk := make([]byte, 64)
	space := len(chunk) - len(header)

	for i := 0; len(apdu) > 0; i++ {
		// Construct the new message to stream
		chunk = append(chunk[:0], header...)
		binary.BigEndian.PutUint16(chunk[3:], uint16(i))

		if len(apdu) > space {
			chunk = append(chunk, apdu[:space]...)
			apdu = apdu[space:]
		} else {
			chunk = append(chunk, apdu...)
			apdu = nil
		}
		// Send over to the device
		log.Trace().Msgf("Ledger: rawCall: Data chunk sent to the Ledger", "chunk", hexutil.Bytes(chunk))

		resp := bus.Fetch("usb", "write", &bus.B_UsbWrite{
			USB_ID: usb_id,
			Data:   chunk,
		})

		if resp.Error != nil {
			log.Error().Err(resp.Error).Msg("Ledger: rawCall: Error writing to device")
			return nil, resp.Error
		}
		// if _, err := w.device.Write(chunk); err != nil {
		// 	return nil, err
		// }
	}
	// Stream the reply back from the wallet in 64 byte chunks
	var reply []byte
	// chunk = chunk[:64] // Yeah, we surely have enough space
	for {
		// Read the next chunk from the Ledger wallet

		resp := bus.Fetch("usb", "read", &bus.B_UsbRead{
			USB_ID: usb_id,
		})

		if resp.Error != nil {
			log.Error().Err(resp.Error).Msg("Ledger: rawCall: Error reading from device")
			return nil, resp.Error
		}

		r, ok := resp.Data.(*bus.B_UsbRead_Response)
		if !ok {
			log.Error().Msg("Ledger: rawCall: Invalid message data")
			return nil, bus.ErrInvalidMessageData
		}
		chunk = r.Data[:64]

		// if _, err := io.ReadFull(w.device, chunk); err != nil {
		// 	return nil, err
		// }
		log.Trace().Msgf("Data chunk received from the Ledger", "chunk", hexutil.Bytes(chunk))

		// Make sure the transport header matches
		if chunk[0] != 0x01 || chunk[1] != 0x01 || chunk[2] != 0x05 {
			log.Error().Msgf("Ledger: rawCall: Invalid reply header: %v", chunk)
			return nil, errors.New("invalid reply header")
		}
		// If it's the first chunk, retrieve the total message length
		var payload []byte

		if chunk[3] == 0x00 && chunk[4] == 0x00 {
			reply = make([]byte, 0, int(binary.BigEndian.Uint16(chunk[5:7])))
			payload = chunk[7:]
		} else {
			payload = chunk[5:]
		}
		// Append to the reply and stop when filled up
		if left := cap(reply) - len(reply); left > len(payload) {
			reply = append(reply, payload...)
		} else {
			reply = append(reply, payload[:left]...)
			break
		}
	}
	return reply[:len(reply)-2], nil
}
