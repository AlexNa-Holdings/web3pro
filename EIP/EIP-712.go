package EIP

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/rs/zerolog/log"
)

type EIP712_TypedData struct {
	Types       map[string][]EIP712_Type `json:"types"`
	PrimaryType string                   `json:"primaryType"`
	Domain      map[string]interface{}   `json:"domain"`
	Message     map[string]interface{}   `json:"message"`
}

// Type represents a field in the TypedData structure
type EIP712_Type struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// U256 encodes a big.Int as a 32-byte array for use as a uint256 in Ethereum.
func U256(n *big.Int) []byte {
	return common.LeftPadBytes(n.Bytes(), 32)
}

func encodeField(fieldType string, value interface{}) ([]byte, error) {

	switch fieldType {
	case "string":
		s, ok := value.(string)
		if !ok {
			log.Error().Msgf("EIP712 encodeField: expected string, got %T", value)
			return nil, fmt.Errorf("expected string, got %T", value)
		}
		return crypto.Keccak256Hash([]byte(s)).Bytes(), nil
	case "uint256":
		bigValue := new(big.Int)

		if v, ok := value.(*big.Int); ok {
			bigValue.Set(v)
		} else if v, ok := value.(float64); ok {
			bigValue.SetInt64(int64(v))
		} else if v, ok := value.(string); ok {
			if strings.HasPrefix(v, "0x") {
				v = v[2:]
				bigValue.SetString(v, 16)
			} else {
				bigValue.SetString(v, 10)
			}
		} else {
			log.Error().Msgf("EIP712 encodeField: expected *big.Int or float64, got %T", value)
			return nil, fmt.Errorf("expected *big.Int or float64, got %T", value)
		}
		return U256(bigValue), nil
	case "address":
		s, ok := value.(string)
		if !ok {
			log.Error().Msgf("EIP712 encodeField: expected string, got %T", value)
			return nil, fmt.Errorf("expected string, got %T", value)
		}
		return common.HexToAddress(s).Bytes(), nil
	default:
		return nil, fmt.Errorf("unsupported field type: %s", fieldType)
	}
}

func hashStruct(primaryType string, types []EIP712_Type, data map[string]interface{}) (common.Hash, error) {
	var sb strings.Builder
	sb.WriteString(primaryType + "(")
	for i, field := range types {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(field.Type + " " + field.Name)
	}
	sb.WriteString(")")

	// Hash the struct type
	structTypeHash := crypto.Keccak256Hash([]byte(sb.String()))

	// Hash the struct fields
	var encodedFieldValues []byte
	for _, field := range types {
		value := data[field.Name]
		encodedValue, err := encodeField(field.Type, value)
		if err != nil {
			return common.Hash{}, err
		}
		encodedFieldValues = append(encodedFieldValues, encodedValue...)
	}

	finalHash := crypto.Keccak256Hash(structTypeHash.Bytes(), encodedFieldValues)
	return finalHash, nil
}

func (data *EIP712_TypedData) EncodeEIP712() ([]byte, error) {
	// Encode the domain separator
	domainSeparator, err := hashStruct("EIP712Domain", data.Types["EIP712Domain"], data.Domain)
	if err != nil {
		return nil, fmt.Errorf("failed to hash domain: %w", err)
	}

	// Encode the message
	messageHash, err := hashStruct(data.PrimaryType, data.Types[data.PrimaryType], data.Message)
	if err != nil {
		return nil, fmt.Errorf("failed to hash message: %w", err)
	}

	// Final hash according to EIP-712
	finalHash := crypto.Keccak256Hash(
		[]byte("\x19\x01"),
		domainSeparator.Bytes(),
		messageHash.Bytes(),
	)

	return finalHash.Bytes(), nil
}
