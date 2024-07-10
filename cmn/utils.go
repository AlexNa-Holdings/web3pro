package cmn

import (
	"fmt"
	"math/big"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/core"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/usb"
	"github.com/ethereum/go-ethereum/common"
)

var Bus *usb.USB
var Core *core.Core

func GetID(info core.EnumerateEntry) (string, error) {

	// switch info.Vendor {
	// case "Ledger":
	// 	// TO DO
	// 	return "12345", nil
	// }
	return info.Path, nil
}

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

// func Amount2Str(amount *big.Int, decimals int) string {
// 	str := amount.String()

// 	if len(str) <= decimals {
// 		str = strings.Repeat("0", decimals-len(str)) + str
// 	}

// 	str = str[:len(str)-decimals] + "." + str[len(str)-decimals:]

// 	str = strings.TrimRight(str, "0")
// 	str = strings.TrimRight(str, ".")
// 	str = strings.TrimLeft(str, "0")
// 	if str == "" || str[0] == '.' {
// 		str = "0" + str
// 	}

// 	return str
// }

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

func FormatDollars(a float64, fixed bool) string {
	bf := big.NewFloat(a)
	bf = bf.Mul(bf, big.NewFloat(10000000000))
	bi, _ := bf.Int(nil)
	r := FormatAmount(bi, 10, fixed, "$")
	return r
}

func FormatDollarsNormal(num float64) string {
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

func FormatUInt64(v uint64, fixed bool, prefix string) string {
	return FormatAmount(new(big.Int).SetUint64(v), 0, fixed, prefix)
}

func FormatAmount(v *big.Int, decimals int, fixed bool, prefix string) string {
	suffixes := []string{"", "K", "M", "B", "T", "Qa", "Qi", "^21", "^24", "^27", "^30", "^33", "^36", "^39",
		"^42", "^45", "^48", "^51", "^54", "^57", "^60", "^63", "^66", "^69", "^72", "^75", "^76"}
	negSuffixes := []string{"", "/K", "/M", "/B", "/T", "/Qa", "/Qi", "/^21", "/^24", "/^27", "/^30", "/^33",
		"/^36", "/^39", "/^42", "/^45", "/^48", "/^51", "/^54", "/^57", "/^60", "/^63", "/^66", "/^69", "/^72", "/^75", "/^76"}

	//if v == 0
	if v.Cmp(big.NewInt(0)) == 0 || decimals < 0 || decimals > 75 {
		if fixed {
			return "  0.00   "
		}
		return "0.00"
	}

	// Convert the big.Int value to a string
	strValue := v.String()

	// Determine the position of the decimal point
	decPos := len(strValue) - decimals

	//Also add 000 to the end
	strValue += "000"

	// Adjust the string to frame the first significant three digits
	if decPos < 0 {
		strValue = strings.Repeat("0", -decPos) + strValue
		decPos = 0
	}

	if decPos%3 != 0 {
		strValue = strings.Repeat("0", 3-decPos%3) + strValue
		decPos += 3 - decPos%3
	}

	exp := 0
	for exp = len(suffixes) - 1; exp >= -(len(suffixes) - 1); exp-- {

		if decPos-exp*3-3 < 0 {
			continue
		}

		if decPos-exp*3-3 >= len(strValue)-3 {
			break
		}

		if strValue[decPos-exp*3-3:decPos-exp*3] != "000" {
			break
		}
	}

	tree := strValue[decPos-exp*3-3 : decPos-exp*3]
	two := strValue[decPos-exp*3 : decPos-exp*3+2]
	suffix := ""
	if exp < 0 {
		suffix = negSuffixes[-exp]
	} else {
		suffix = suffixes[exp]
	}

	if tree[0] == '0' && tree[1] == '0' {
		tree = tree[2:]
	} else if tree[0] == '0' {
		tree = tree[1:]
	}

	if fixed {
		fixed_l := 3 + len(prefix)
		s := prefix + tree
		for len(s) < fixed_l {
			s = " " + s
		}

		return fmt.Sprintf("%s.%s%-3s", s, two, suffix)

	} else {

		return fmt.Sprintf("%s%s.%s%s", prefix, tree, two, suffix)
	}

}

func (t *Token) Str2Value(str string) (*big.Int, error) {
	s := strings.TrimSpace(str)
	if s == "" {
		return nil, fmt.Errorf("empty string")
	}

	//check that only digits and max one dot
	dot_index := -1
	for i, c := range s {
		if c == '.' {
			if dot_index != -1 {
				return nil, fmt.Errorf("Two dots in the string")
			} else {
				dot_index = i
				continue
			}
		}
		if c < '0' || c > '9' {
			return nil, fmt.Errorf("invalid character: %c", c)
		}
	}

	// remove dot
	if dot_index != -1 {
		s = s[:dot_index] + s[dot_index+1:]
	} else {
		dot_index = len(s)
	}

	n_after_dot := len(s) - dot_index
	if n_after_dot < t.Decimals {
		s += strings.Repeat("0", t.Decimals-n_after_dot)
	} else {
		s = s[dot_index+t.Decimals:]
	}

	value, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return nil, fmt.Errorf("invalid number: %s", s)
	}

	return value, nil
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
