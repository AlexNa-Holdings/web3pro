package cmn

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/pbkdf2"
)

const SOLT_SIZE = 32

var CurrentWallet *Wallet

func Open(name string, pass string) error {

	w, err := openFromFile(DataFolder+"/wallets/"+name, pass)

	if err == nil {
		w.filePath = DataFolder + "/wallets/" + name
		w.password = pass
		w.writeMutex = sync.Mutex{}

		w.AuditNativeTokens()
		if w.CurrentChain == "" || w.GetBlockchain(w.CurrentChain) == nil {
			if len(w.Blockchains) > 0 {
				w.CurrentChain = w.Blockchains[0].Name
			} else {
				w.CurrentChain = ""
			}
		}
		if w.CurrentAddress == (common.Address{}) || w.GetAddress(w.CurrentAddress.String()) == nil {
			if len(w.Addresses) > 0 {
				w.CurrentAddress = w.Addresses[0].Address
			} else {
				w.CurrentAddress = common.Address{}
			}
		}

		if w.CurrentOrigin == "" || w.GetOrigin(w.CurrentOrigin) == nil {
			if len(w.Origins) > 0 {
				w.CurrentOrigin = w.Origins[0].URL
			} else {
				w.CurrentOrigin = ""
			}
		}

		CurrentWallet = w
		bus.Send("wallet", "open", nil)
	}
	return err
}

func (w *Wallet) DeleteBlockchain(name string) error {
	w.writeMutex.Lock()
	defer w.writeMutex.Unlock()

	bd := w.GetBlockchain(name)
	if bd == nil {
		return errors.New("blockchain not found")
	}

	for i, b := range w.Blockchains {
		if b.Name == bd.Name {
			w.Blockchains = append(w.Blockchains[:i], w.Blockchains[i+1:]...)
			break
		}
	}

	for i, t := range w.Tokens {
		if t.Blockchain == name {
			w.Tokens = append(w.Tokens[:i], w.Tokens[i+1:]...)
		}
	}

	for i, o := range w.LP_V3_Providers {
		if o.ChainId == bd.ChainId {
			w.LP_V3_Providers = append(w.LP_V3_Providers[:i], w.LP_V3_Providers[i+1:]...)
		}
	}

	w.AuditNativeTokens()

	return w.Save()
}

func (w *Wallet) AuditNativeTokens() {

	eddited := false

	for _, b := range w.Blockchains { // audit native tokens
		found := false
		for _, t := range w.Tokens {
			if t.Blockchain == b.Name && t.Native {
				t.Symbol = b.Currency
				found = true
				break
			}
		}
		if !found {
			w.Tokens = append(w.Tokens, &Token{
				Blockchain: b.Name,
				Name:       b.Currency,
				Symbol:     b.Currency,
				Decimals:   18,
				Native:     true,
			})
			eddited = true
		}
	}

	for _, b := range w.Blockchains { // audit wrapped native tokens
		if b.WTokenAddress != (common.Address{}) {
			wt := w.GetTokenByAddress(b.Name, b.WTokenAddress)
			if wt == nil {

				nt, _ := w.GetNativeToken(b)

				w.Tokens = append(w.Tokens, &Token{
					Blockchain:     b.Name,
					Address:        b.WTokenAddress,
					Name:           "Wrapped " + b.Currency,
					Symbol:         "W" + b.Currency,
					Decimals:       18,
					Native:         false,
					PriceFeeder:    nt.PriceFeeder,
					PriceFeedParam: nt.PriceFeedParam,
				})
				eddited = true
			}
		}
	}

	to_remove := []int{}
	for i, t := range w.Tokens {
		if t.Native && w.GetBlockchain(t.Blockchain) == nil {
			to_remove = append([]int{i}, to_remove...)
			break
		}
	}

	for _, i := range to_remove {
		w.Tokens = append(w.Tokens[:i], w.Tokens[i+1:]...)
		eddited = true
	}

	w.MarkUniqueTokens()

	if eddited {
		w.Save()
	}
}

func (w *Wallet) GetBlockchain(n string) *Blockchain {
	for _, b := range w.Blockchains {
		if b.Name == n {
			return b
		}
	}

	// try to find by chain id
	chain_id, err := strconv.Atoi(n)
	if err == nil {
		for _, b := range w.Blockchains {
			if b.ChainId == chain_id {
				return b
			}
		}
	}

	return nil
}

func (w *Wallet) GetBlockchainById(id int) *Blockchain {
	for _, b := range w.Blockchains {
		if b.ChainId == id {
			return b
		}
	}
	return nil
}

func (w *Wallet) Save() error {
	return SaveToFile(w, w.filePath, w.password)
}

func Exists(name string) bool {
	_, err := os.Stat(DataFolder + "/wallets/" + name)
	return !os.IsNotExist(err)
}

func Create(name, pass string) error {
	w := &Wallet{}

	return SaveToFile(w, DataFolder+"/wallets/"+name, pass)
}

func (w *Wallet) RemoveOrigin(url string) error {

	w.writeMutex.Lock()
	deleted := false
	for i, o := range w.Origins {
		if o.URL == url {
			w.Origins = append(w.Origins[:i], w.Origins[i+1:]...)
			deleted = true
			break
		}
	}

	if w.CurrentOrigin == url {
		w.CurrentOrigin = ""
	}

	w.writeMutex.Unlock()

	if deleted {
		bus.Send("wallet", "origin-changed", url)
		return w.Save()
	}

	return errors.New("origin not found")
}

func (w *Wallet) GetOrigin(url string) *Origin {
	for _, o := range w.Origins {
		if o.URL == url {
			return o
		}
	}
	return nil
}

func (w *Wallet) AddOrigin(o *Origin) error {
	if w.GetOrigin(o.URL) != nil {
		log.Error().Msgf("Origin already exists: %s\n", o.URL)
		return fmt.Errorf("origin already exists: %s", o.URL)
	}

	o.URL = strings.TrimSpace(o.URL)

	// check the URL format
	if o.URL == "" {
		return errors.New("origin URL is empty")
	}
	_, err := url.Parse(o.URL)
	if err != nil {
		return err
	}

	w.writeMutex.Lock()
	w.Origins = append(w.Origins, o)
	if w.CurrentOrigin == "" {
		w.CurrentOrigin = o.URL
	}
	w.writeMutex.Unlock()

	bus.Send("wallet", "origin-changed", o.URL)
	return w.Save()
}

func (w *Wallet) RemoveOriginAddress(url string, a string) error {
	o := w.GetOrigin(url)
	if o == nil {
		return errors.New("origin not found")
	}

	addr := w.GetAddressByName(a)
	if addr == nil {
		return errors.New("address not found")
	}

	w.writeMutex.Lock()
	for i, na := range o.Addresses {
		if na == addr.Address {
			o.Addresses = append(o.Addresses[:i], o.Addresses[i+1:]...)
			break
		}
	}
	w.writeMutex.Unlock()

	if len(o.Addresses) == 0 {
		w.RemoveOrigin(url)
	}

	bus.Send("wallet", "origin-addresses-changed", url)
	return w.Save()
}

func (o *Origin) IsAllowed(a common.Address) bool {
	for _, aa := range o.Addresses {
		if aa == a {
			return true
		}
	}
	return false
}

func (w *Wallet) AddOriginAddress(url string, a string) error {
	o := w.GetOrigin(url)
	if o == nil {
		return errors.New("origin not found")
	}

	addr := w.GetAddressByName(a)
	if addr == nil {
		return errors.New("address not found")
	}

	found := false
	w.writeMutex.Lock()
	for _, na := range o.Addresses {
		if na == addr.Address {
			found = true
			break
		}
	}
	if !found {
		o.Addresses = append(o.Addresses, addr.Address)
	}
	w.writeMutex.Unlock()

	bus.Send("wallet", "origin-addresses-changed", url)

	return w.Save()
}

func (w *Wallet) SetOriginChain(url string, ch string) error {
	o := w.GetOrigin(url)
	if o == nil {
		return errors.New("origin not found")
	}

	b := w.GetBlockchain(ch)
	if b == nil {
		return errors.New("blockchain not found")
	}

	o.ChainId = b.ChainId

	bus.Send("wallet", "origin-chain-changed", url)

	return w.Save()
}

func (w *Wallet) SetOrigin(url string) error {
	o := w.GetOrigin(url)
	if o == nil {
		return errors.New("origin not found")
	}

	w.CurrentOrigin = url

	return w.Save()
}

func (w *Wallet) PromoteOriginAddress(url string, a string) error {
	o := w.GetOrigin(url)
	if o == nil {
		return errors.New("origin not found")
	}

	addr := w.GetAddressByName(a)
	if addr == nil {
		return errors.New("address not found")
	}

	w.writeMutex.Lock()
	found := false
	for i, na := range o.Addresses {
		if na == addr.Address {
			found = true
			if i == 0 {
				break
			}
			o.Addresses = append([]common.Address{addr.Address}, append(o.Addresses[:i], o.Addresses[i+1:]...)...)
			break
		}
	}
	w.writeMutex.Unlock()

	if !found {
		return errors.New("address not found")
	}

	bus.Send("wallet", "origin-addresses-changed", url)

	return w.Save()
}

func WalletList() []string {
	files, err := os.ReadDir(DataFolder + "/wallets")
	if err != nil {
		log.Error().Msgf("Error reading directory: %v\n", err)
		return nil
	}

	names := []string{}
	for _, file := range files {
		names = append(names, file.Name())
	}

	return names
}

func encrypt(data []byte, passphrase []byte) ([]byte, error) {
	block, err := aes.NewCipher([]byte(passphrase))
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

func decrypt(data []byte, passphrase []byte) ([]byte, error) {
	block, err := aes.NewCipher([]byte(passphrase))
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, err
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// GenerateKey derives a key from a password using PBKDF2
func generateKey(password string, salt []byte) []byte {
	return pbkdf2.Key([]byte(password), salt, 4096, 32, sha256.New)
}

func SaveToFile(w *Wallet, file, pass string) error {
	w.writeMutex.Lock()
	defer w.writeMutex.Unlock()

	jsonData, err := json.Marshal(w)
	if err != nil {
		log.Error().Msgf("Error marshaling JSON: %v\n", err)
		return err
	}

	solt := make([]byte, SOLT_SIZE)
	_, err = rand.Read(solt)
	if err != nil {
		log.Error().Msgf("Error generating salt: %v\n", err)
		return err
	}

	key := generateKey(pass, solt)

	encrypted, err := encrypt(jsonData, key)
	if err != nil {
		log.Error().Msgf("Error encrypting data: %v\n", err)
		return err
	}

	// write with the soltencrypted
	err = os.WriteFile(file, append(solt, encrypted...), 0644)
	if err != nil {
		log.Error().Msgf("Error writing file: %v\n", err)
		return err
	}

	bus.Send("wallet", "saved", nil)
	return nil
}

func openFromFile(file string, pass string) (*Wallet, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		log.Error().Msgf("Error reading file: %v\n", err)
		return nil, err
	}

	solt := data[:SOLT_SIZE]
	data = data[SOLT_SIZE:]

	key := generateKey(pass, solt)

	decrypted, err := decrypt(data, key)
	if err != nil {
		log.Error().Msgf("Error decrypting data: %v\n", err)
		return nil, err
	}

	w := &Wallet{}
	err = json.Unmarshal(decrypted, w)
	if err != nil {
		log.Error().Msgf("Error unmarshaling JSON: %v\n", err)
		return nil, err
	}

	if w.Contracts == nil {
		w.Contracts = make(map[common.Address]*Contract)
	}

	if w.Tokens == nil {
		w.Tokens = []*Token{}
	}

	if w.Addresses == nil {
		w.Addresses = []*Address{}
	}

	if w.Signers == nil {
		w.Signers = []*Signer{}
	}

	if w.Origins == nil {
		w.Origins = []*Origin{}
	}

	if w.Blockchains == nil {
		w.Blockchains = []*Blockchain{}
	}

	return w, nil
}

func (w *Wallet) GetSigner(n string) *Signer {
	for _, s := range w.Signers {
		if s.Name == n {
			return s
		}
	}
	return nil
}

func (w *Wallet) GetSignerWithCopyIndex(name string) (*Signer, int) {
	for _, s := range w.Signers {
		for j, c := range s.Copies {
			if c == name {
				return s, j
			}
		}
	}
	return nil, -1
}

func (w *Wallet) GetSignerWithCopies(name string) []*Signer {
	res := []*Signer{}

	s := w.GetSigner(name)
	if s == nil {
		return res
	}

	res = append(res, s)

	for _, c := range s.Copies {
		s = w.GetSigner(c)
		if s != nil {
			res = append(res, s)
		}
	}

	return res
}

func (w *Wallet) GetAddress(a any) *Address {
	switch a := a.(type) {
	case string:
		// normalize the format
		if common.IsHexAddress(a) {
			a = common.HexToAddress(a).String()
		} else {
			return nil
		}

		for _, s := range w.Addresses {
			if s.Address.String() == a {
				return s
			}
		}
	case common.Address:
		for _, s := range w.Addresses {
			if s.Address == a {
				return s
			}
		}
	}

	return nil
}

func (w *Wallet) GetContract(a common.Address) *Contract {
	c, ok := w.Contracts[a]

	if !ok {
		return nil
	}
	return c
}

func (w *Wallet) GetAddressByName(n string) *Address {
	for _, s := range w.Addresses {
		if s.Name == n {
			return s
		}
	}

	return w.GetAddress(n)
}

func (w *Wallet) GetToken(b string, a string) *Token {
	if common.IsHexAddress(a) {
		t := w.GetTokenByAddress(b, common.HexToAddress(a))
		if t != nil {
			return t
		}
	}

	return w.GetTokenBySymbol(b, a)
}

func (w *Wallet) GetNativeToken(b *Blockchain) (*Token, error) {
	for _, t := range w.Tokens {
		if t.Blockchain == b.Name && t.Native {
			return t, nil
		}
	}

	log.Error().Msgf("Native token not found for blockchain %s", b.Name)
	return nil, errors.New("native token not found")
}

func (w *Wallet) AddToken(b string, a common.Address, n string, s string, d int) error {
	if w.GetTokenByAddress(b, a) != nil {
		return errors.New("token already exists")
	}

	t := &Token{
		Blockchain: b,
		Address:    a,
		Name:       n,
		Symbol:     s,
		Decimals:   d,
	}

	w.Tokens = append(w.Tokens, t)
	w.MarkUniqueTokens()

	return w.Save()
}

func (w *Wallet) GetTokenByAddress(b string, a common.Address) *Token {
	if a.Cmp(common.Address{}) == 0 {
		b := w.GetBlockchain(b)
		if b == nil {
			return nil
		}
		t, err := w.GetNativeToken(b)
		if err != nil {
			return nil
		}
		return t
	}

	// for _, t := range w.Tokens {
	// 	if t.Blockchain == b && ((!t.Native && t.Address == a) ||
	// 		(t.Native && a == common.Address{})) {
	// 		return t
	// 	}
	// }

	for _, t := range w.Tokens {
		if t.Blockchain == b && (t.Address == a) {
			return t
		}
	}

	return nil
}

func (w *Wallet) GetTokenBySymbol(b string, s string) *Token {
	for _, t := range w.Tokens {
		if t.Blockchain == b && t.Symbol == s {
			if !t.Unique {
				return nil // Ambiguous
			}
			return t
		}
	}
	return nil
}

func (w *Wallet) DeleteToken(b string, a common.Address) {
	for i, t := range w.Tokens {
		if t.Blockchain == b && t.Address == a {
			w.Tokens = append(w.Tokens[:i], w.Tokens[i+1:]...)
			return
		}
	}
}

func (w *Wallet) MarkUniqueTokens() {
	for _, b := range w.Blockchains {
		for i, t := range w.Tokens {
			if t.Blockchain == b.Name {
				t.Unique = true
				for j := 0; j < i; j++ {
					if w.Tokens[j].Blockchain == b.Name && w.Tokens[j].Symbol == t.Symbol {
						t.Unique = false
						w.Tokens[j].Unique = false
						break
					}
				}
			}
		}
	}
}

func (w *Wallet) IsContractTrusted(addr common.Address) bool {
	if w.Contracts[addr] == nil {
		return false
	}
	return w.Contracts[addr].Trusted
}

func (w *Wallet) TrustContract(addr common.Address) error {
	if w.Contracts[addr] == nil {
		w.Contracts[addr] = &Contract{}
	}
	w.Contracts[addr].Trusted = true

	return w.Save()
}

func (w *Wallet) UntrustContract(addr common.Address) error {
	if w.Contracts[addr] == nil {
		return nil
	}
	w.Contracts[addr].Trusted = false

	return w.Save()
}

func (w *Wallet) AddLP_V3(lp *LP_V3) error {
	if w.GetLP_V3(lp.ChainId, lp.Provider) != nil {
		return errors.New("provider already exists")
	}

	if w.GetLP_V3_by_name(lp.ChainId, lp.Name) != nil {
		return errors.New("provider with the same name already exists")
	}

	w.LP_V3_Providers = append(w.LP_V3_Providers, lp)
	return w.Save()
}

func (w *Wallet) GetLP_V3(chainId int, addr common.Address) *LP_V3 {
	for _, lp := range w.LP_V3_Providers {
		if lp.ChainId == chainId && lp.Provider == addr {
			return lp
		}
	}
	return nil
}

func (w *Wallet) GetLP_V3_by_name(chainId int, name string) *LP_V3 {
	for _, lp := range w.LP_V3_Providers {
		if lp.ChainId == chainId && lp.Name == name {
			return lp
		}
	}
	return nil
}

func (w *Wallet) RemoveLP_V3(chainId int, provider common.Address) error {
	for i, lp := range w.LP_V3_Providers {
		if lp.ChainId == chainId && lp.Provider == provider {
			w.LP_V3_Providers = append(w.LP_V3_Providers[:i], w.LP_V3_Providers[i+1:]...)

			// remove all positions
			for j := len(w.LP_V3_Positions) - 1; j >= 0; j-- {
				if w.LP_V3_Positions[j].ChainId == chainId && w.LP_V3_Positions[j].Provider == provider {
					w.LP_V3_Positions = append(w.LP_V3_Positions[:j], w.LP_V3_Positions[j+1:]...)
				}
			}

			return w.Save()
		}
	}

	return errors.New("provider not found")
}

func (w *Wallet) AddLP_V3Position(lp *LP_V3_Position) error {

	if pos := w.GetLP_V3Position(lp.Owner, lp.ChainId, lp.Provider, lp.NFT_Token); pos != nil {
		// update
		pos.Token0 = lp.Token0
		pos.Token1 = lp.Token1
		pos.Pool = lp.Pool
		pos.Fee = lp.Fee
	}

	w.LP_V3_Positions = append(w.LP_V3_Positions, lp)
	return w.Save()
}

func (w *Wallet) GetLP_V3Position(addr common.Address, chainId int, provider common.Address, nft *big.Int) *LP_V3_Position {
	for _, lp := range w.LP_V3_Positions {
		if lp.Owner.Cmp(addr) == 0 && lp.ChainId == chainId &&
			lp.Provider.Cmp(provider) == 0 &&
			lp.NFT_Token.Cmp(nft) == 0 {
			return lp
		}
	}
	return nil
}

func (w *Wallet) RemoveLP_V3Position(addr common.Address, chainId int, provider common.Address, nft *big.Int) error {
	for i, lp := range w.LP_V3_Positions {
		if lp.Owner.Cmp(addr) == 0 && lp.ChainId == chainId && lp.Provider == provider && lp.NFT_Token.Cmp(nft) == 0 {
			w.LP_V3_Positions = append(w.LP_V3_Positions[:i], w.LP_V3_Positions[i+1:]...)
			return w.Save()
		}
	}
	return errors.New("position not found")
}
