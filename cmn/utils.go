package cmn

import (
	"fmt"
	"math/big"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
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
	return fmt.Sprintf("<l text:'%s' action:'copy %s' tip:'Copy %s'>", sh, sa, sa)
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
