package EIP

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
