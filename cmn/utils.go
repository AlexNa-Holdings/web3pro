package cmn

import (
	"fmt"
	"math/big"
	"net/url"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/AlexNa-Holdings/web3pro/EIP"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func Contains(s string, subststr string) bool {
	return strings.Contains(
		strings.ToLower(s),
		strings.ToLower(subststr),
	)
}

// Split splits the input string into a slice of strings. Guarantees that the
// result has at least 3 elements.
func Split(input string) []string {
	return SplitN(input, 3)
}

// Split splits the input string into a slice of strings. Guarantees that the
// result has at least n elements.
func SplitN(input string, n int) []string {
	re := regexp.MustCompile(`'[^']*'|"[^"]*"|\b[^'\s"]+\b`)
	matches := re.FindAllString(input, -1)

	var result []string
	for _, match := range matches {
		// Remove quotes if the match is a quoted string
		if (match[0] == '"' && match[len(match)-1] == '"') || (match[0] == '\'' && match[len(match)-1] == '\'') {
			match = match[1 : len(match)-1]
		}
		result = append(result, match)
	}

	// make sure the result has at least 3 elements
	for len(result) < n {
		result = append(result, "")
	}

	return result
}

func IsInArray(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func Amount2Str(amount *big.Int, decimals int) string {
	str := amount.String()

	if len(str) <= decimals {
		str = strings.Repeat("0", decimals-len(str)+1) + str
	}

	str = str[:len(str)-decimals] + "." + str[len(str)-decimals:]

	// Trim trailing zeros and the decimal point if necessary
	str = strings.TrimRight(str, "0")
	str = strings.TrimRight(str, ".")

	// Add commas to the integer part
	parts := strings.Split(str, ".")
	intPart := parts[0]
	fracPart := ""
	if len(parts) > 1 {
		fracPart = parts[1]
	}

	intPartWithCommas := ""
	n := len(intPart)
	for i, ch := range intPart {
		if i > 0 && (n-i)%3 == 0 {
			intPartWithCommas += ","
		}
		intPartWithCommas += string(ch)
	}

	if fracPart != "" {
		return intPartWithCommas + "." + fracPart
	}
	return intPartWithCommas
}

func FmtFloat64(v float64, fixed bool) string {
	n := NewXF_Float64(v)
	return n.Format(fixed, "")
}

func FmtFloat64D(v float64, fixed bool) string {
	n := NewXF_Float64(v)
	return n.Format(fixed, "$")
}

func (t *Token) FmtValue64D(v float64, fixed bool) string {
	n := NewXF_Float64(v)
	return n.Format(fixed, "$")
}

func FormatUInt64(v uint64, fixed bool) string {
	n := NewXF_UInt64(v)
	return n.Format(fixed, "")
}

func FmtAmount(amount *big.Int, decimals int, fixed bool) string {
	n := &XF{Int: new(big.Int).Set(amount), decimals: decimals}
	return n.Format(fixed, "")
}

// convert string to the "weis" value
func (t *Token) Str2Wei(str string) (*big.Int, error) {
	return Str2Wei(str, t.Decimals)
}

// convert string to the "weis" value
func Str2Wei(str string, decimals int) (*big.Int, error) {
	val, err := ParseXF(str)
	if err != nil {
		log.Error().Err(err).Msg("Str2Value parse error")
		return nil, err
	}
	val.Mul(Pow10(decimals))
	return val.BigInt(), nil
}

// Normal format with 4 decimal places
func FmtFloat64DN(num float64) string {
	decimalPlaces := 4
	// Format the number with the specified decimal places
	format := fmt.Sprintf("%%.%df", decimalPlaces)
	numberStr := fmt.Sprintf(format, num)

	// Split the formatted number into integer and fractional parts
	parts := strings.Split(numberStr, ".")
	intPart := parts[0]
	fracPart := ""
	if len(parts) > 1 {
		fracPart = parts[1]
	}

	// Insert commas as thousand separators in the integer part
	intPartWithCommas := ""
	n := len(intPart)
	for i, ch := range intPart {
		if i > 0 && (n-i)%3 == 0 {
			intPartWithCommas += ","
		}
		intPartWithCommas += string(ch)
	}

	// Combine integer and fractional parts
	if fracPart != "" {
		return "$" + intPartWithCommas + "." + fracPart
	}
	return "$" + intPartWithCommas
}

func ShortAddress(a common.Address) string {
	s := a.String()
	return s[:6] + gocui.ICON_3DOTS + s[len(s)-4:]
}

func (t *Token) Value2Str(value *big.Int) string {
	return Amount2Str(value, t.Decimals)
}

func AddressShortLinkTag(a common.Address) string {
	sa := a.String()
	sh := ShortAddress(a)
	return fmt.Sprintf("<l text:'%s' action:'copy %s' tip:'%s'>", sh, sa, sa)
}

func OpenBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	default:
		return fmt.Errorf("unsupported platform")
	}

	return exec.Command(cmd, args...).Start()
}

func (t *Token) GetPrintName() string {
	if t.Native {
		return "(native)"
	}
	return t.Symbol
}

func (b *Blockchain) ExplorerLink(address common.Address) string {
	if b.ExplorerUrl == "" {
		return ""
	}
	if strings.HasSuffix(b.ExplorerUrl, "/") {
		return b.ExplorerUrl + "address/" + address.Hex()
	}

	return b.ExplorerUrl + "/address/" + address.Hex()

}

func (t *Token) Float64(value *big.Int) float64 {
	return Float64(value, t.Decimals)
}

func Float64(value *big.Int, decimals int) float64 {
	f := new(big.Float).SetInt(value)
	f = f.Quo(f, new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)))
	r, _ := f.Float64()
	return r
}

func Float(value *big.Int, decimals int) *big.Float {
	f := new(big.Float).SetInt(value)
	f = f.Quo(f, new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)))
	return f
}

func AddAddressLink(v *gocui.View, a common.Address) {
	v.AddLink(a.String(), "copy "+a.String(), "Copy address", "")
}

func AddAddressShortLink(v *gocui.View, a common.Address) {
	s := a.String()
	v.AddLink(s[:6]+gocui.ICON_3DOTS+s[len(s)-4:], "copy "+a.String(), a.String(), "")
}

func TagLink(text, action, tip string) string {
	return fmt.Sprintf("<l text:\"%s\" action:\"%s\" tip:\"%s\">", text, action, tip)
}

func TagAddressShortLink(a common.Address) string {
	s := a.String()

	return fmt.Sprintf("<l text:'%s%s%s' action:'copy %s' tip:'%s'>",
		s[:6], gocui.ICON_3DOTS, s[len(s)-4:], a.String(), a.String())
}

func AddValueLink(v *gocui.View, val *big.Int, t *Token) {
	if v == nil {
		return
	}

	if t == nil {
		return
	}

	xf := NewXF(val, t.Decimals)

	text := FmtAmount(val, t.Decimals, true)
	v.AddLink(text, "copy "+xf.String(), xf.String(), "")
}

func TagValueLink(val *big.Int, t *Token) string {
	if t == nil {
		return ""
	}

	xf := NewXF(val, t.Decimals)

	return fmt.Sprintf("<l text:'%s' action:'copy %s' tip:'%s'>", FmtAmount(val, t.Decimals, true), xf.String(), xf.String())
}

func AddDollarValueLink(v *gocui.View, val *big.Int, t *Token) {
	if v == nil {
		return
	}

	if t == nil {
		return
	}

	xf := NewXF(val, t.Decimals)

	f := t.Price * xf.Float64()

	text := FmtFloat64D(f, true)
	n := fmt.Sprintf("%.15f", f)

	v.AddLink(text, "copy "+n, n, "")
}

func TagDollarValueLink(val *big.Int, t *Token) string {
	if t == nil {
		return ""
	}

	xf := NewXF(val, t.Decimals)
	f := t.Price * xf.Float64()

	return fmt.Sprintf("<l text:'%s' action:'copy %f' tip:'%f'>", FmtFloat64D(f, true), f, f)
}

func TagShortDollarValueLink(val *big.Int, t *Token) string {
	if t == nil {
		return ""
	}

	xf := NewXF(val, t.Decimals)
	f := t.Price * xf.Float64()

	return fmt.Sprintf("<l text:'%s' action:'copy %f' tip:'%f'>", FmtFloat64D(f, false), f, f)
}

func TagShortDollarLink(val float64) string {
	return fmt.Sprintf("<l text:'%s' action:'copy %.15f' tip:'%.15f'>", FmtFloat64D(val, false), val, val)
}

func AddValueSymbolLink(v *gocui.View, val *big.Int, t *Token) {
	if v == nil {
		return
	}

	if t == nil {
		return
	}

	xf := NewXF(val, t.Decimals)
	text := FmtAmount(val, t.Decimals, true)

	v.AddLink(text, "copy "+xf.String(), xf.String(), "")
}

func TagValueSymbolLink(val *big.Int, t *Token) string {
	if t == nil {
		return ""
	}

	xf := NewXF(val, t.Decimals)

	return fmt.Sprintf("<l text:'%s' action:'copy %s' tip:'%s'> %s",
		FmtAmount(val, t.Decimals, true), xf.String(), xf.String(), t.Symbol)
}

func TagShortValueSymbolLink(val *big.Int, t *Token) string {
	if t == nil {
		return ""
	}

	xf := NewXF(val, t.Decimals)

	return fmt.Sprintf("<l text:'%s' action:'copy %s' tip:'%s'> %s",
		FmtAmount(val, t.Decimals, false), xf.String(), xf.String(), t.Symbol)
}

func GetHostName(URL string) string {
	parsedURL, err := url.Parse(URL)
	if err != nil {
		return URL
	}
	return parsedURL.Hostname()
}

func IntFromAny(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case string:
		if strings.HasPrefix(v, "0x") {
			v = v[2:]
			i, _ := new(big.Int).SetString(v, 16)
			return int(i.Int64())
		} else {
			i, _ := new(big.Int).SetString(v, 10)
			return int(i.Int64())
		}
	default:
		return 0
	}
}

func AddressFromAny(value any) common.Address {
	switch v := value.(type) {
	case string:
		return common.HexToAddress(v)
	default:
		return common.Address{}
	}
}

func BigIntFromAny(value any) *big.Int {
	switch v := value.(type) {
	case *big.Int:
		return v
	case float64:
		return new(big.Int).SetInt64(int64(v))
	case string:
		if strings.HasPrefix(v, "0x") {
			v = v[2:]
			i, _ := new(big.Int).SetString(v, 16)
			return i
		} else {
			i, _ := new(big.Int).SetString(v, 10)
			return i
		}
	default:
		return nil
	}
}

type SignedDataInfo struct {
	Type       string
	Blockchain *Blockchain
	Token      *Token
	Value      *big.Int
	Address    *Address
}

func ConfirmEIP712Template(data *EIP.EIP712_TypedData) string {
	var sb strings.Builder
	var info SignedDataInfo
	titleCaser := cases.Title(language.English)
	w := CurrentWallet

	info.Type = data.PrimaryType

	if w != nil {
		if data.PrimaryType == "Permit" {
			// collect all info about the permit

			chain_id := IntFromAny(data.Domain["chainId"])
			info.Blockchain = w.GetBlockchainById(chain_id)
			ta := AddressFromAny(data.Domain["verifyingContract"])
			if info.Blockchain != nil {
				info.Token = w.GetTokenByAddress(info.Blockchain.Name, ta)
			}
			info.Value = BigIntFromAny(data.Message["value"])
			owner := AddressFromAny(data.Message["owner"])
			info.Address = w.GetAddress(owner.String())
		}
	}

	// Primary Type
	sb.WriteString(fmt.Sprintf("<b>Types: </b>%s\n", data.PrimaryType))

	// Format Domain
	sb.WriteString("<line text:Domain>\n")
	for _, field := range data.Types["EIP712Domain"] {
		value := data.Domain[field.Name]
		formattedValue := formatFieldValue(info, field.Name, field.Type, value)
		sb.WriteString(fmt.Sprintf("<b>%s: </b>%s\n", titleCaser.String(field.Name), formattedValue))
	}

	// Format Message
	sb.WriteString("<line text:Message>\n")
	for _, field := range data.Types[data.PrimaryType] {
		value := data.Message[field.Name]
		formattedValue := formatFieldValue(info, field.Name, field.Type, value)
		sb.WriteString(fmt.Sprintf("<b>%s: </b>%s\n", titleCaser.String(field.Name), formattedValue))
	}

	sb.WriteString(`
<c><button text:"Sign" id:ok> <button text:"Cancel">`)

	return sb.String()
}

func formatFieldValue(info SignedDataInfo, fieldName, fieldType string, value interface{}) string {
	s := "?"

	switch fieldType {
	case "string":
		if sv, ok := value.(string); ok {
			s = sv
		}
	case "uint256":
		if v, ok := value.(*big.Int); ok {
			s = v.String()
		} else if v, ok := value.(float64); ok {
			s = fmt.Sprintf("%d", uint64(v))
		} else if v, ok := value.(string); ok {
			s = v
		}

	case "address":
		if sv, ok := value.(string); ok {
			s = TagAddressShortLink(common.HexToAddress(sv))
		}
	}

	if info.Type == "Permit" {
		switch fieldName {
		case "value":
			if info.Value != nil && info.Token != nil {
				s = TagShortValueSymbolLink(info.Value, info.Token)
			}
		case "owner":
			if info.Address != nil {
				s = TagAddressShortLink(info.Address.Address) + " " + info.Address.Name
			}
		case "deadline":
			dl := IntFromAny(value)
			t := time.Unix(0, int64(dl)*int64(time.Millisecond))
			s = t.Format("2006-01-02 15:04:05 MST")
		case "chainId":
			if info.Blockchain != nil {
				s = fmt.Sprintf("%d %s", info.Blockchain.ChainId, info.Blockchain.Name)
			}
		}
	}
	return s
}
