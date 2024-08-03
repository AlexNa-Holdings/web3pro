package ledger

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/rs/zerolog/log"
)

var StatusCodes = map[int]string{
	0x5102: "NOT_ENOUGH_SPACE",
	0x5123: "APP_NOT_FOUND_OR_INVALID_CONTEXT",
	0x5419: "GEN_AES_KEY_FAILED",
	0x541a: "INTERNAL_CRYPTO_OPERATION_FAILED",
	0x541b: "INTERNAL_COMPUTE_AES_CMAC_FAILED",
	0x541c: "ENCRYPT_APP_STORAGE_FAILED",
	0x5501: "USER_REFUSED_ON_DEVICE",
	0x5502: "PIN_NOT_SET",
	0x5515: "LOCKED_DEVICE",
	0x6300: "GP_AUTH_FAILED",
	0x63c0: "PIN_REMAINING_ATTEMPTS",
	0x6511: "WRONG APP",
	0x6611: "DEVICE_NOT_ONBOARDED_2",
	0x662e: "CUSTOM_IMAGE_EMPTY",
	0x662f: "DEVICE_IN_RECOVERY_MODE",
	0x6642: "INVALID_BACKUP_STATE",
	0x6643: "INVALID_RESTORE_STATE",
	0x6700: "INCORRECT_LENGTH",
	0x670a: "INVALID_APP_NAME_LENGTH",
	0x6733: "INVALID_BACKUP_LENGTH",
	0x6734: "INVALID_CHUNK_LENGTH",
	0x684a: "INVALID_BACKUP_HEADER",
	0x6800: "MISSING_CRITICAL_PARAMETER",
	0x6981: "COMMAND_INCOMPATIBLE_FILE_STRUCTURE",
	0x6982: "SECURITY_STATUS_NOT_SATISFIED",
	0x6985: "CONDITIONS_OF_USE_NOT_SATISFIED",
	0x6a80: "INCORRECT_DATA",
	0x6a84: "NOT_ENOUGH_MEMORY_SPACE",
	0x6a88: "REFERENCED_DATA_NOT_FOUND",
	0x6a89: "FILE_ALREADY_EXISTS",
	0x6b00: "INCORRECT_P1_P2",
	0x6d00: "INS_NOT_SUPPORTED",
	0x6d02: "UNKNOWN_APDU",
	0x6d07: "DEVICE_NOT_ONBOARDED",
	0x6e00: "CLA_NOT_SUPPORTED",
	0x6f00: "TECHNICAL_PROBLEM",
	0x6f42: "LICENSING",
	0x6faa: "HALTED",
	0x9000: "OK",
	0x9240: "MEMORY_PROBLEM",
	0x9400: "NO_EF_SELECTED",
	0x9402: "INVALID_OFFSET",
	0x9404: "FILE_NOT_FOUND",
	0x9408: "INCONSISTENT_FILE",
	0x9484: "ALGORITHM_NOT_SUPPORTED",
	0x9485: "INVALID_KCV",
	0x9802: "CODE_NOT_INITIALIZED",
	0x9804: "ACCESS_CONDITION_NOT_FULFILLED",
	0x9808: "CONTRADICTION_SECRET_CODE_STATUS",
	0x9810: "CONTRADICTION_INVALIDATION",
	0x9840: "CODE_BLOCKED",
	0x9850: "MAX_VALUE_REACHED",
}

func rawCall(usb_id string, apdu *APDU, data []byte, hail *bus.B_Hail, hail_delay int) ([]byte, error) {
	// Construct the message payload, possibly split into multiple chunks
	buf := make([]byte, 2, 7+len(data))

	binary.BigEndian.PutUint16(buf, uint16(5+len(data)))
	buf = append(buf, []byte{apdu.cla, apdu.op_code, apdu.p1, apdu.p2, byte(len(data))}...)
	buf = append(buf, data...)

	// Stream all the chunks to the device
	header := []byte{0x01, 0x01, 0x05, 0x00, 0x00} // Channel ID and command tag appended
	chunk := make([]byte, 64)
	space := len(chunk) - len(header)

	for i := 0; len(buf) > 0; i++ {
		// Construct the new message to stream
		chunk = append(chunk[:0], header...)
		binary.BigEndian.PutUint16(chunk[3:], uint16(i))

		if len(buf) > space {
			chunk = append(chunk, buf[:space]...)
			buf = buf[space:]
		} else {
			chunk = append(chunk, buf...)
			buf = nil
		}
		// Send over to the device
		log.Trace().Msgf("Ledger: rawCall: Writing data chunk to the Ledger: %s", hexutil.Bytes(chunk))

		resp := bus.Fetch("usb", "write", &bus.B_UsbWrite{
			USB_ID: usb_id,
			Data:   chunk,
		})

		if resp.Error != nil {
			log.Error().Err(resp.Error).Msg("Ledger: rawCall: Error writing to device")
			return nil, resp.Error
		}
	}
	// Stream the reply back from the wallet in 64 byte chunks
	var reply []byte
	first_read := true
	for {
		// Read the next chunk from the Ledger wallet

		var resp *bus.Message
		if first_read && hail != nil {
			resp = bus.FetchWithHail("usb", "read", &bus.B_UsbRead{
				USB_ID: usb_id,
			}, hail, hail_delay)
			first_read = false
		} else {
			resp = bus.Fetch("usb", "read", &bus.B_UsbRead{
				USB_ID: usb_id,
			})
		}

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

		log.Trace().Msgf("Data chunk received from the Ledger: %s", hexutil.Bytes(chunk))

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

	rc := int(binary.BigEndian.Uint16(reply[len(reply)-2:]))
	if rc != 0x9000 {
		if msg, ok := StatusCodes[rc]; ok {
			return nil, fmt.Errorf("Ledger: rawCall: Error response from device: %s", msg)
		}
		return nil, fmt.Errorf("Ledger: rawCall: Error response from device: %x", rc)
	}

	log.Debug().Msgf("Ledger: rawCall: Reply: %s", hexutil.Bytes(reply))

	return reply[:len(reply)-2], nil
}

// UnwrapResponseAPDU parses a response of 64 byte packets into the real data
func UnwrapResponseAPDU(channel uint16, pipe <-chan []byte, packetSize int) ([]byte, error) {
	var sequenceIdx uint16

	var totalResult []byte
	var totalSize uint16
	var done = false

	// return values from DeserializePacket
	var result []byte
	var responseSize uint16
	var err error

	foundZeroSequence := false
	isSequenceZero := false

	for !done {
		// Read next packet from the channel
		buffer := <-pipe

		result, responseSize, isSequenceZero, err = DeserializePacket(channel, buffer, sequenceIdx) // this may fail if the wrong sequence arrives (espeically if left over all 0000 was in the buffer from the last tx)
		if err != nil {
			return nil, err
		}

		// Recover from a known error condition:
		// * Discard messages left over from previous exchange until isSequenceZero == true
		if foundZeroSequence == false && isSequenceZero == false {
			continue
		}
		foundZeroSequence = true

		// Initialize totalSize (previously we did this if sequenceIdx == 0, but sometimes Nano X can provide the first sequenceIdx == 0 packet with all zeros, then a useful packet with sequenceIdx == 1
		if totalSize == 0 {
			totalSize = responseSize
		}

		buffer = buffer[packetSize:]
		totalResult = append(totalResult, result...)
		sequenceIdx++

		if len(totalResult) >= int(totalSize) {
			done = true
		}
	}

	// Remove trailing zeros
	totalResult = totalResult[:totalSize]
	return totalResult, nil
}

func DeserializePacket(
	channel uint16,
	buffer []byte,
	sequenceIdx uint16) (result []byte, totalResponseLength uint16, isSequenceZero bool, err error) {

	isSequenceZero = false

	if (sequenceIdx == 0 && len(buffer) < 7) || (sequenceIdx > 0 && len(buffer) < 5) {
		return nil, 0, isSequenceZero, fmt.Errorf("cannot deserialize the packet. header information is missing")
	}

	var headerOffset uint8

	if codec.Uint16(buffer) != channel {
		return nil, 0, isSequenceZero, fmt.Errorf("invalid channel.  expected %d, got: %d", channel, codec.Uint16(buffer))
	}
	headerOffset += 2

	if buffer[headerOffset] != 0x05 {
		return nil, 0, isSequenceZero, fmt.Errorf("invalid tag.  expected %d, got: %d", 0x05, buffer[headerOffset])
	}
	headerOffset++

	foundSequenceIdx := codec.Uint16(buffer[headerOffset:])
	if foundSequenceIdx == 0 {
		isSequenceZero = true
	} else {
		isSequenceZero = false
	}

	if foundSequenceIdx != sequenceIdx {
		return nil, 0, isSequenceZero, fmt.Errorf("wrong sequenceIdx.  expected %d, got: %d", sequenceIdx, foundSequenceIdx)
	}
	headerOffset += 2

	if sequenceIdx == 0 {
		totalResponseLength = codec.Uint16(buffer[headerOffset:])
		headerOffset += 2
	}

	result = make([]byte, len(buffer)-int(headerOffset))
	copy(result, buffer[headerOffset:])

	return result, totalResponseLength, isSequenceZero, nil
}
