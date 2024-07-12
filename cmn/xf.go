package cmn

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/rs/zerolog/log"
)

// eXtraodinary float
// including eXtraordinary formatting
// by AlexNa

var FMT_SUFFIXES = []string{"", "K", "M", "B", "T", "Qa", "Qi", "^21", "^24", "^27", "^30", "^33", "^36", "^39",
	"^42", "^45", "^48", "^51", "^54", "^57", "^60", "^63", "^66", "^69", "^72", "^75", "^76"}
var FMT_NEG_SUFFIXES = []string{"", "/K", "/M", "/B", "/T", "/Qa", "/Qi", "/^21", "/^24", "/^27", "/^30", "/^33",
	"/^36", "/^39", "/^42", "/^45", "/^48", "/^51", "/^54", "/^57", "/^60", "/^63", "/^66", "/^69", "/^72", "/^75", "/^76"}

const MAX_DECIMALS = 80
const PRECISION = 20

type XF struct {
	*big.Int
	decimals int
	NaN      bool
}

var XF_NaN = &XF{big.NewInt(0), 0, true}

func NewXF(v *big.Int, decimal int) *XF {
	return &XF{v, decimal, false}
}

func NewXF_Float64(v float64) *XF {
	n, _ := ParseXF(fmt.Sprintf("%f", v))
	return n
}

func NewXF_BigFloat(v *big.Float) *XF {
	n, err := ParseXF(v.Text('f', 100))
	if err != nil {
		log.Debug().Err(err).Msg("NewXF_BigFloat")
		return XF_NaN
	}

	return n
}

func (n *XF) String() string {
	if n.NaN {
		return "NaN"
	}

	s := n.Int.String()
	if len(s) <= n.decimals {
		s = strings.Repeat("0", n.decimals-len(s)+1) + s
	}
	s = s[:len(s)-n.decimals] + "." + s[len(s)-n.decimals:]
	return s
}

func NewXF_UInt64(v uint64) *XF {
	n := new(big.Int).SetUint64(v)
	return NewXF(n, 0)
}

func (n *XF) OrderOfMagnitude() int {
	return len(n.String()) - n.decimals
}

func (n *XF) BigInt() *big.Int {
	n.Norm()

	r := new(big.Int).Set(n.Int)

	if n.decimals > 0 {
		r.Div(n.Int, Pow10(n.decimals).Int)
	}
	return r
}

func (n *XF) IsZero() bool {
	return n.Int.Cmp(big.NewInt(0)) == 0
}

var PD []XF

func Pow10(N int) *XF {
	log.Debug().Msgf("Pow10: N: %d", N)

	if PD == nil {
		n := NewXF_UInt64(1)
		d10 := NewXF_UInt64(10)
		for i := 0; i < 100; i++ {
			PD = append(PD, XF{
				Int:      new(big.Int).Set(n.Int),
				decimals: n.decimals,
				NaN:      false,
			})
			n.Mul(d10)
		}
	}

	if N >= 0 {
		if N < len(PD) {
			return &PD[N]
		} else {
			return XF_NaN
		}
	} else {
		if -N < len(PD) {
			return &XF{big.NewInt(1), -N, false}
		} else {
			return XF_NaN
		}
	}
}

func (n *XF) Norm() *XF {
	if !n.NaN {
		if n.decimals < 0 {
			n.Int.Mul(n.Int, Pow10(-n.decimals).Int)
			n.decimals = 0
		}

		if n.decimals > MAX_DECIMALS {
			n.Int.Div(n.Int, Pow10(n.decimals-MAX_DECIMALS).Int)
			n.decimals = MAX_DECIMALS
		}
	}
	return n
}

func (n *XF) Dump() string {
	if n.NaN {
		return "NaN"
	} else {
		return fmt.Sprintf("Int: %s, Decimals: %d", n.Int.String(), n.decimals)
	}
}

func ParseXF(s string) (*XF, error) {
	log.Debug().Msgf("ParseXF: %s", s)

	s = strings.TrimSpace(s)
	suffix_found := false
	m := NewXF_UInt64(1)

	for i := 1; i < len(FMT_NEG_SUFFIXES); i++ {
		if strings.HasSuffix(s, FMT_NEG_SUFFIXES[i]) {
			m = Pow10(-i * 3)
			s = strings.TrimSuffix(s, FMT_NEG_SUFFIXES[i])

			log.Debug().Msgf("Suggix: %s", FMT_NEG_SUFFIXES[i])
			log.Debug().Msgf("S: %s", s)
			log.Debug().Msgf("M: %s", m.Dump())

			suffix_found = true
			break
		}
	}

	if !suffix_found {
		for i := 1; i < len(FMT_SUFFIXES); i++ {
			if strings.HasSuffix(s, FMT_SUFFIXES[i]) {
				m = Pow10(i * 3)
				s = strings.TrimSuffix(s, FMT_SUFFIXES[i])
				break
			}
		}
	}

	dotPos := -1
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			if dotPos != -1 {
				return nil, errors.New("multiple dots in number")
			}
			dotPos = i
			break
		}
	}

	decimals := 0
	if dotPos != -1 {
		m.decimals += len(s) - dotPos - 1
		s = s[:dotPos] + s[dotPos+1:] // remove dot
	}

	num, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return nil, errors.New("invalid number")
	}

	return NewXF(num, decimals).Mul(m).Norm(), nil
}

func (n *XF) Mul(x *XF) *XF {
	if n.NaN || x.NaN {
		n.NaN = true
		return n
	}
	n.Int.Mul(n.Int, x.Int)
	n.decimals += x.decimals
	return n
}

func Mul(x, y *XF) *XF {
	if x.NaN || y.NaN {
		return &XF{big.NewInt(0), 0, true}
	}

	return NewXF(new(big.Int).Mul(x.Int, y.Int), x.decimals+y.decimals)
}

func (n *XF) Add(x *XF) *XF {
	if n.NaN || x.NaN {
		n.NaN = true
		return n
	}

	if n.decimals > x.decimals {
		x = x.Mul(Pow10(n.decimals - x.decimals))
	} else if n.decimals < x.decimals {
		n = n.Mul(Pow10(x.decimals - n.decimals))
	}
	n.Int.Add(n.Int, x.Int)
	return n
}

func Add(x, y *XF) *XF {
	if x.NaN || y.NaN {
		return &XF{big.NewInt(0), 0, true}
	}

	return NewXF(new(big.Int).Add(x.Int, y.Int), x.decimals)
}

func (n *XF) Sub(x *XF) *XF {
	if n.NaN || x.NaN {
		n.NaN = true
		return n
	}

	if n.decimals > x.decimals {
		x = x.Mul(Pow10(n.decimals - x.decimals))
	} else if n.decimals < x.decimals {
		n = n.Mul(Pow10(x.decimals - n.decimals))
	}
	n.Int.Sub(n.Int, x.Int)
	return n
}

func Sub(x, y *XF) *XF {
	if x.NaN || y.NaN {
		return &XF{big.NewInt(0), 0, true}
	}

	return NewXF(new(big.Int).Sub(x.Int, y.Int), x.decimals)
}

func (n *XF) Div(x *XF) *XF {
	if n.NaN || x.NaN {
		n.NaN = true
		return n
	}

	log.Debug().Msgf("Div: %s / %s", n.Dump(), x.Dump())

	nm := n.OrderOfMagnitude()
	xm := x.OrderOfMagnitude() //TODO?????

	factor := 1
	if nm < xm+PRECISION {
		factor = xm + PRECISION - nm
		n = n.Mul(Pow10(factor))
	}

	n.Int.Quo(n.Int, x.Int)

	n.decimals = n.decimals - x.decimals + factor
	n.Norm()

	log.Debug().Msgf("Div res: %s ", n.Dump())

	return n
}

func Div(x, y *XF) *XF {
	if x.NaN || y.NaN {
		return &XF{big.NewInt(0), 0, true}
	}

	return NewXF(new(big.Int).Div(x.Int, y.Int), x.decimals-y.decimals)
}

func (n *XF) Format(fixed bool, prefix string) string {
	if n.NaN {
		if fixed {
			return "  NaN   "
		} else {
			return "NaN"
		}
	}

	//if v == 0
	if n.Int.Cmp(big.NewInt(0)) == 0 || n.decimals < 0 || n.decimals > MAX_DECIMALS {
		if fixed {
			return "  0.00   "
		}
		return "0.00"
	}

	// Convert the big.Int value to a string
	strValue := n.Int.String()

	// Determine the position of the decimal point
	decPos := len(strValue) - n.decimals

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
	for exp = len(FMT_SUFFIXES) - 1; exp >= -(len(FMT_SUFFIXES) - 1); exp-- {

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
		suffix = FMT_NEG_SUFFIXES[-exp]
	} else {
		suffix = FMT_SUFFIXES[exp]
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
