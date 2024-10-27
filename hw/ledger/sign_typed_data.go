package ledger

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ava-labs/coreth/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/rs/zerolog/log"
)

func signTypedData_v4(msg *bus.Message) (string, error) {
	m, _ := msg.Data.(*bus.B_SignerSignTypedData_v4)

	ledger := provide_device(m.Name)
	if ledger == nil {
		return "", fmt.Errorf("no device found with name %s", m.Name)
	}

	err := provide_eth_app(ledger.USB_ID, "Ethereum")
	if err != nil {
		return "", err
	}

	// Send the struct definition payloads
	err = SendStructDefinition(msg, ledger, m.TypedData)
	if err != nil {
		log.Error().Err(err).Msgf("SignTypedData: Error building struct definition payloads: %s", m.Path)
		return "", err
	}

	// Send the struct implementation payloads
	err = SendStructImplementation(msg, ledger, m.TypedData)
	if err != nil {
		log.Error().Err(err).Msgf("SignTypedData: Error building struct implementation payloads: %s", m.Path)
		return "", err
	}

	var payload []byte

	dp, err := accounts.ParseDerivationPath(m.Path)
	if err != nil {
		log.Error().Err(err).Msgf("SignTypedData: Error parsing path: %s", m.Path)
		return "", err
	}
	payload = append(payload, serializePath(dp)...)

	_, data, err := apitypes.TypedDataAndHash(m.TypedData)
	if err != nil {
		log.Error().Msgf("SignTypedData: Failed to hash typed data: %v", err)
		return "", err
	}

	payload = append(payload, data[2:34]...)
	payload = append(payload, data[34:66]...)

	log.Trace().Msgf("LEDGER: SIGN TYPED DATA: PATH %s", m.Path)

	save_mode := ledger.Pane.Mode
	save_template := ledger.Pane.GetTemplate()
	defer func() {
		ledger.Pane.SetTemplate(save_template)
		ledger.Pane.SetMode(save_mode)
	}()

	ledger.Pane.SetTemplate("<w><c>\n<blink>" + cmn.ICON_ALERT + "</blink>Please sign the message on the device\n")
	ledger.Pane.SetMode("template")

	reply, err := call(ledger.USB_ID, &SIGN_MSG_APDU, payload)
	if err != nil {
		log.Error().Err(err).Msgf("SignTypedData: Error signing typed data: %s", m.Path)
		return "", err
	}

	var sig []byte
	sig = append(sig, reply[1:]...) // R + S
	sig = append(sig, reply[0])     // V

	log.Debug().Msgf("Signature: 0x%x", sig)
	log.Debug().Msgf("Len: %d", len(sig))

	if len(sig) == 65 && (sig[64] == 0 || sig[64] == 1) {
		sig[64] += 27
	}

	return fmt.Sprintf("0x%x", sig), nil
}

func SendStructFieldDef(msg *bus.Message, ledger *Ledger, field apitypes.Type) error {
	// Build TypeDesc byte
	var typeDesc byte = 0x00

	isArray := strings.HasSuffix(field.Type, "[]")
	baseTypeStr := field.Type
	if isArray {
		typeDesc |= 0x80 // TypeArray bit
		baseTypeStr = strings.TrimSuffix(field.Type, "[]")
	}

	// Map base type to Type value
	var typeValue byte
	var hasTypeSize bool
	var typeSize int

	switch {
	case baseTypeStr == "address":
		typeValue = 0x03
	case baseTypeStr == "bool":
		typeValue = 0x04
	case baseTypeStr == "string":
		typeValue = 0x05
	case baseTypeStr == "bytes":
		typeValue = 0x07 // dynamic-sized bytes
	case strings.HasPrefix(baseTypeStr, "bytes"):
		sizeStr := strings.TrimPrefix(baseTypeStr, "bytes")
		size, err := strconv.Atoi(sizeStr)
		if err != nil {
			log.Error().Err(err).Msgf("SendStructFieldDef: Error parsing bytes size: %s", field.Type)
			return fmt.Errorf("invalid bytes size in type %s", field.Type)
		}
		typeValue = 0x06 // fixed-sized bytes
		hasTypeSize = true
		typeSize = size
	case strings.HasPrefix(baseTypeStr, "int"), strings.HasPrefix(baseTypeStr, "uint"):
		intType := baseTypeStr
		if strings.HasPrefix(baseTypeStr, "int") {
			typeValue = 0x01
			intType = strings.TrimPrefix(baseTypeStr, "int")
		} else {
			typeValue = 0x02
			intType = strings.TrimPrefix(baseTypeStr, "uint")
		}
		size, err := strconv.Atoi(intType)
		if err != nil {
			log.Error().Err(err).Msgf("SendStructFieldDef: Error parsing int size: %s", field.Type)
			return fmt.Errorf("invalid int size in type %s", field.Type)
		}
		hasTypeSize = true
		typeSize = size
	default:
		// Assume custom type
		typeValue = 0x00 // custom
	}

	typeDesc |= typeValue & 0x0F

	if hasTypeSize && typeValue != 0x00 {
		typeDesc |= 0x40 // TypeSize bit
	}

	data := []byte{typeDesc}

	// TypeName (if custom type)
	if typeValue == 0x00 {
		typeNameBytes := []byte(baseTypeStr)
		data = append(data, byte(len(typeNameBytes)))
		data = append(data, typeNameBytes...)
	}

	// TypeSize (if applicable)
	if hasTypeSize {
		data = append(data, byte(typeSize/8)) // size in bytes
	}

	// ArrayLevelCount and ArrayLevels (if array)
	if isArray {
		data = append(data, byte(1)) // ArrayLevelCount
		// Assuming dynamic array for simplicity
		data = append(data, byte(0)) // 0 for dynamic array
	}

	// KeyNameLength and KeyName
	keyNameBytes := []byte(field.Name)
	data = append(data, byte(len(keyNameBytes)))
	data = append(data, keyNameBytes...)

	log.Trace().Msgf("LEDGER: STRUCT DEFINITION: FIELD %s", field.Name)

	_, err := call(ledger.USB_ID, &STRUCT_DEF_FIELD, data)
	if err != nil {
		log.Error().Err(err).Msgf("SendStructFieldDef: Error setting struct field: %s", field.Name)
		return err
	}

	return nil
}

func GetTypeOrder(typedData apitypes.TypedData) []string {
	visited := make(map[string]bool)
	order := []string{}

	var visit func(string)
	visit = func(typeName string) {
		if visited[typeName] {
			return
		}
		visited[typeName] = true

		for _, field := range typedData.Types[typeName] {
			// Check if the field type is a custom type
			baseType := strings.TrimSuffix(field.Type, "[]")
			if _, exists := typedData.Types[baseType]; exists {
				visit(baseType)
			}
		}
		order = append(order, typeName)
	}

	// Start by visiting the domain type if it exists
	if _, exists := typedData.Types["EIP712Domain"]; exists {
		visit("EIP712Domain")
	}

	// Then visit the primary type
	visit(typedData.PrimaryType)

	// Reverse the order to get the correct sequence
	for i, j := 0, len(order)-1; i < j; i, j = i+1, j-1 {
		order[i], order[j] = order[j], order[i]
	}

	return order
}

func SendStructDefinition(msg *bus.Message, ledger *Ledger, typedData apitypes.TypedData) error {
	typeOrder := GetTypeOrder(typedData)

	for _, typeName := range typeOrder {
		fields := typedData.Types[typeName]

		log.Trace().Msgf("LEDGER: STRUCT DEFINITION: Name %s", typeName)

		_, err := call(ledger.USB_ID, &STRUCT_DEF_NAME, []byte(typeName))
		if err != nil {
			log.Error().Err(err).Msgf("SignTypedData: Error setting struct name: %s", typeName)
			return err
		}

		// Build and append field payloads
		for _, field := range fields {
			err := SendStructFieldDef(msg, ledger, field)
			if err != nil {
				log.Error().Err(err).Msgf("SignTypedData: Error sending struct field payload: %s", field.Name)
				return err
			}
		}
	}

	return nil
}

func SendRootStruct(msg *bus.Message, ledger *Ledger, rootStructName string) error {

	log.Trace().Msgf("LEDGER: ROOT STRUCT: Name %s", rootStructName)

	_, err := call(ledger.USB_ID, &STRUCT_IMPL_ROOT, []byte(rootStructName))
	if err != nil {
		log.Error().Err(err).Msgf("SignTypedData: Error setting root struct: %s", rootStructName)
		return err
	}

	return nil
}

func SendFieldValue(msg *bus.Message, ledger *Ledger, value interface{}, typeStr string) error {
	var valueBytes []byte

	switch {
	case typeStr == "address":
		addrStr, ok := value.(string)
		if !ok {
			log.Error().Msgf("SendFieldValue: Expected string for address type")
			return fmt.Errorf("expected string for address type")
		}
		addr := common.HexToAddress(addrStr)
		valueBytes = addr.Bytes()

	case typeStr == "bool":
		boolVal, ok := value.(bool)
		if !ok {
			log.Error().Msgf("SendFieldValue: Expected bool for bool type")
			return fmt.Errorf("expected bool for bool type")
		}
		if boolVal {
			valueBytes = []byte{1}
		} else {
			valueBytes = []byte{0}
		}

	case typeStr == "string":
		strVal, ok := value.(string)
		if !ok {
			log.Error().Msgf("SendFieldValue: Expected string for string type")
			return fmt.Errorf("expected string for string type")
		}
		valueBytes = []byte(strVal)

	case typeStr == "bytes":
		bytesVal, ok := value.(string)
		if !ok {
			log.Error().Msgf("SendFieldValue: Expected string for bytes type")
			return fmt.Errorf("expected string for bytes type")
		}
		valueBytes = common.FromHex(bytesVal)

	case strings.HasPrefix(typeStr, "bytes"):
		// Fixed-size bytes
		bytesVal, ok := value.(string)
		if !ok {
			log.Error().Msgf("SendFieldValue: Expected string for bytes type")
			return fmt.Errorf("expected string for bytes type")
		}
		valueBytes = common.FromHex(bytesVal)

	case strings.HasPrefix(typeStr, "int"), strings.HasPrefix(typeStr, "uint"):
		// Integer types
		var bigIntVal *big.Int
		switch v := value.(type) {
		case string:
			bigIntVal = new(big.Int)
			_, ok := bigIntVal.SetString(v, 10)
			if !ok {
				log.Error().Msgf("SendFieldValue: Invalid integer value: %s", v)
				return fmt.Errorf("invalid integer value: %s", v)
			}
		case float64:
			bigIntVal = big.NewInt(int64(v))
		case int:
			bigIntVal = big.NewInt(int64(v))
		case int64:
			bigIntVal = big.NewInt(v)
		case uint64:
			bigIntVal = new(big.Int).SetUint64(v)
		case *big.Int:
			bigIntVal = v
		case *math.HexOrDecimal256:
			bigIntVal = (*big.Int)(v)
		default:
			log.Error().Msgf("SendFieldValue: Unsupported value type for int: %T", value)
			return fmt.Errorf("unsupported value type for int: %T", value)
		}

		// Ensure the integer is serialized as a 32-byte big-endian value
		valueBytes = bigIntVal.FillBytes(make([]byte, 32))

	default:
		log.Error().Msgf("SendFieldValue: Unsupported type: %s", typeStr)
		return fmt.Errorf("unsupported type: %s", typeStr)
	}

	// Build the data: Value length (2 bytes BE) + Value
	data := make([]byte, 2+len(valueBytes))
	binary.BigEndian.PutUint16(data[0:2], uint16(len(valueBytes)))
	copy(data[2:], valueBytes)

	log.Trace().Msgf("LEDGER: FIELD VALUE: Type %s", typeStr)

	_, err := call(ledger.USB_ID, &STRUCT_IMPL_FIELD, data)
	if err != nil {
		log.Error().Err(err).Msgf("SendFieldValue: Error sending field value payload: %s", typeStr)
		return err
	}

	return nil
}

func SendStructImplementation(msg *bus.Message, ledger *Ledger, typedData apitypes.TypedData) error {
	// 1. Send the root struct for the domain
	err := SendRootStruct(msg, ledger, "EIP712Domain")
	if err != nil {
		log.Error().Err(err).Msg("Error sending root struct for EIP712Domain")
		return err
	}

	// 2. Send the field values for the domain
	domainValues := typedData.Domain.Map()
	domainFields := typedData.Types["EIP712Domain"]
	for _, field := range domainFields {
		value, exists := domainValues[field.Name]
		if !exists {
			return fmt.Errorf("missing value for domain field %s", field.Name)
		}
		err := SendFieldValue(msg, ledger, value, field.Type)
		if err != nil {
			log.Error().Err(err).Msgf("error sending value for domain field %s", field.Name)
			return err
		}
	}

	// 3. Send the root struct for the primary type (message)
	err = SendRootStruct(msg, ledger, typedData.PrimaryType)
	if err != nil {
		log.Error().Err(err).Msgf("SignTypedData: Error setting root struct: %s", typedData.PrimaryType)
		return err
	}

	// 4. Send the message payloads for the primary type
	err = SendMessagePayloads(msg, ledger, typedData, typedData.PrimaryType, typedData.Message)
	if err != nil {
		log.Error().Err(err).Msgf("SignTypedData: Error sending message payloads: %s", typedData.PrimaryType)
		return err
	}

	return nil
}

func SendMessagePayloads(msg *bus.Message, ledger *Ledger, typedData apitypes.TypedData, typeName string, messageData map[string]interface{}) error {
	fields := typedData.Types[typeName]
	for _, field := range fields {
		value := messageData[field.Name]

		// Determine if the field is a custom type
		baseType := strings.TrimSuffix(field.Type, "[]")
		isCustomType := false
		if _, exists := typedData.Types[baseType]; exists {
			isCustomType = true
		}

		// Handle arrays
		isArray := strings.HasSuffix(field.Type, "[]")

		if isCustomType {
			if isArray {
				// Handle array of custom types
				arrayValues, ok := value.([]interface{})
				if !ok {
					log.Error().Msgf("SendMessagePayloads: Expected array for field %s", field.Name)
					return fmt.Errorf("expected array for field %s", field.Name)
				}

				// Build array payload
				arrayPayload := make([]byte, 1)
				arrayPayload[0] = byte(len(arrayValues))

				_, err := call(ledger.USB_ID, &STRUCT_IMPL_ARRAY, arrayPayload)
				if err != nil {
					log.Error().Err(err).Msgf("SendMessagePayloads: Error setting struct array: %s", field.Name)
					return err
				}

				// Build payloads for each element
				for _, elem := range arrayValues {
					// Start new struct
					err = SendRootStruct(msg, ledger, baseType)
					if err != nil {
						log.Error().Err(err).Msgf("SendMessagePayloads: Error setting root struct: %s", baseType)
						return err
					}

					elemMap, ok := elem.(map[string]interface{})
					if !ok {
						log.Error().Msgf("SendMessagePayloads: Expected object for field %s", field.Name)
						return fmt.Errorf("invalid element type for field %s", field.Name)
					}

					err = SendMessagePayloads(msg, ledger, typedData, baseType, elemMap)
					if err != nil {
						log.Error().Err(err).Msgf("SendMessagePayloads: Error sending message payloads: %s", baseType)
						return err
					}
				}
			} else {
				// Custom type
				err := SendRootStruct(msg, ledger, baseType)
				if err != nil {
					log.Error().Err(err).Msgf("SendMessagePayloads: Error setting root struct: %s", baseType)
					return err
				}

				nestedMessage, ok := value.(map[string]interface{})
				if !ok {
					log.Error().Msgf("SendMessagePayloads: Expected object for field %s", field.Name)
					return fmt.Errorf("expected object for field %s", field.Name)
				}

				err = SendMessagePayloads(msg, ledger, typedData, baseType, nestedMessage)
				if err != nil {
					log.Error().Err(err).Msgf("SendMessagePayloads: Error sending message payloads: %s", baseType)
					return err
				}
			}
		} else {
			if isArray {
				// Handle array of base types
				arrayValues, ok := value.([]interface{})
				if !ok {
					log.Error().Msgf("SendMessagePayloads: Expected array for field %s", field.Name)
					return fmt.Errorf("expected array for field %s", field.Name)
				}

				// Build array payload
				arrayPayload := make([]byte, 1)
				arrayPayload[0] = byte(len(arrayValues))

				_, err := call(ledger.USB_ID, &STRUCT_IMPL_ARRAY, arrayPayload)
				if err != nil {
					log.Error().Err(err).Msgf("SendMessagePayloads: Error setting struct array: %s", field.Name)
					return err
				}

				// Build payloads for each element
				for _, elem := range arrayValues {
					err = SendFieldValue(msg, ledger, elem, baseType)
					if err != nil {
						log.Error().Err(err).Msgf("SendMessagePayloads: Error sending field value payload: %s", field.Name)
						return err
					}
				}
			} else {
				// Base type
				err := SendFieldValue(msg, ledger, value, field.Type)
				if err != nil {
					log.Error().Err(err).Msgf("SendMessagePayloads: Error sending field value payload: %s", field.Name)
					return err
				}
			}
		}
	}

	return nil
}
