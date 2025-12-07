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
	if err != nil {
		return err
	}

	w.writeMutex.Lock()
	defer w.writeMutex.Unlock()

	if w != nil {
		w._locked_AuditNativeTokens()
		if w.CurrentChainId == 0 || w.GetBlockchain(w.CurrentChainId) == nil {
			if len(w.Blockchains) > 0 {
				w.CurrentChainId = w.Blockchains[0].ChainId
			} else {
				w.CurrentChainId = 0
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
	return nil
}

func (w *Wallet) DeleteBlockchain(name string) error {
	w.writeMutex.Lock()
	defer w.writeMutex.Unlock()

	bd := w.GetBlockchainByName(name)
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
		if t.ChainId == bd.ChainId {
			w.Tokens = append(w.Tokens[:i], w.Tokens[i+1:]...)
		}
	}

	for i, o := range w.LP_V3_Providers {
		if o.ChainId == bd.ChainId {
			w.LP_V3_Providers = append(w.LP_V3_Providers[:i], w.LP_V3_Providers[i+1:]...)
		}
	}
	w._locked_AuditNativeTokens()

	return w._locked_Save()
}

func (w *Wallet) _locked_AuditNativeTokens() {

	eddited := false

	// Migrate missing ShortNames and Multicall addresses from predefined chains
	for _, b := range w.Blockchains {
		for _, predefined := range PredefinedBlockchains {
			if predefined.ChainId == b.ChainId {
				if b.ShortName == "" && predefined.ShortName != "" {
					b.ShortName = predefined.ShortName
					eddited = true
				}
				if b.Multicall == (common.Address{}) && predefined.Multicall != (common.Address{}) {
					b.Multicall = predefined.Multicall
					eddited = true
				}
				break
			}
		}
	}

	// remove doubles
	for i, t := range w.Tokens {
		for j, tt := range w.Tokens {
			if i != j && t.ChainId == tt.ChainId && t.Symbol == tt.Symbol {
				w.Tokens = append(w.Tokens[:j], w.Tokens[j+1:]...)
				eddited = true
			}
		}
	}

	for _, b := range w.Blockchains { // audit native tokens
		found := false
		for _, t := range w.Tokens {
			if t.ChainId == b.ChainId && t.Native {
				t.Symbol = b.Currency
				found = true
				break
			}
		}
		if !found {
			w.Tokens = append(w.Tokens, &Token{
				ChainId:  b.ChainId,
				Name:     b.Currency,
				Symbol:   b.Currency,
				Decimals: 18,
				Native:   true,
			})
			eddited = true
		}
	}

	for _, b := range w.Blockchains { // audit wrapped native tokens
		if b.WTokenAddress != (common.Address{}) {
			wt := w.GetTokenByAddress(b.ChainId, b.WTokenAddress)
			if wt == nil {

				nt, _ := w.GetNativeToken(b)

				w.Tokens = append(w.Tokens, &Token{
					ChainId:        b.ChainId,
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
		if t.Native && w.GetBlockchain(t.ChainId) == nil {
			to_remove = append([]int{i}, to_remove...)
			break
		}
	}

	for _, i := range to_remove {
		w.Tokens = append(w.Tokens[:i], w.Tokens[i+1:]...)
		eddited = true
	}

	w._locked_MarkUniqueTokens()

	if eddited {
		w._locked_Save()
	}
}

func (w *Wallet) GetBlockchainByName(n string) *Blockchain {
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

func (w *Wallet) GetBlockchain(id int) *Blockchain {
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

func (w *Wallet) _locked_Save() error {
	return _locked_SaveToFile(w, w.filePath, w.password)
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
	defer w.writeMutex.Unlock()

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

	if deleted {
		bus.Send("wallet", "origin-changed", url)
		return w._locked_Save()
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
	w.writeMutex.Lock()
	defer w.writeMutex.Unlock()

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

	w.Origins = append(w.Origins, o)
	if w.CurrentOrigin == "" {
		w.CurrentOrigin = o.URL
	}

	bus.Send("wallet", "origin-changed", o.URL)
	return w._locked_Save()
}

func (w *Wallet) RemoveOriginAddress(url string, a common.Address) error {
	w.writeMutex.Lock()
	defer w.writeMutex.Unlock()

	o := w.GetOrigin(url)
	if o == nil {
		return errors.New("origin not found")
	}

	for i, na := range o.Addresses {
		if na == a {
			o.Addresses = append(o.Addresses[:i], o.Addresses[i+1:]...)
			break
		}
	}

	if len(o.Addresses) == 0 {
		w.RemoveOrigin(url)
	}

	bus.Send("wallet", "origin-addresses-changed", url)
	return w._locked_Save()
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
	w.writeMutex.Lock()
	defer w.writeMutex.Unlock()

	o := w.GetOrigin(url)
	if o == nil {
		return errors.New("origin not found")
	}

	addr := w.GetAddressByName(a)
	if addr == nil {
		return errors.New("address not found")
	}

	found := false
	for _, na := range o.Addresses {
		if na == addr.Address {
			found = true
			break
		}
	}
	if !found {
		o.Addresses = append(o.Addresses, addr.Address)
	}

	bus.Send("wallet", "origin-addresses-changed", url)

	return w._locked_Save()
}

func (w *Wallet) SetOriginChain(url string, ch string) error {
	w.writeMutex.Lock()
	defer w.writeMutex.Unlock()

	o := w.GetOrigin(url)
	if o == nil {
		return errors.New("origin not found")
	}

	b := w.GetBlockchainByName(ch)
	if b == nil {
		return errors.New("blockchain not found")
	}

	o.ChainId = b.ChainId

	bus.Send("wallet", "origin-chain-changed", url)

	return w._locked_Save()
}

func (w *Wallet) SetOrigin(url string) error {
	w.writeMutex.Lock()
	defer w.writeMutex.Unlock()

	o := w.GetOrigin(url)
	if o == nil {
		return errors.New("origin not found")
	}

	w.CurrentOrigin = url

	return w._locked_Save()
}

func (w *Wallet) PromoteOriginAddress(url string, a string) error {
	w.writeMutex.Lock()
	defer w.writeMutex.Unlock()

	o := w.GetOrigin(url)
	if o == nil {
		return errors.New("origin not found")
	}

	addr := w.GetAddressByName(a)
	if addr == nil {
		return errors.New("address not found")
	}

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

	if !found {
		return errors.New("address not found")
	}

	bus.Send("wallet", "origin-addresses-changed", url)

	return w._locked_Save()
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

	return _locked_SaveToFile(w, file, pass)
}

func _locked_SaveToFile(w *Wallet, file, pass string) error {

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

	w := &Wallet{
		filePath:   file,
		password:   pass,
		writeMutex: sync.Mutex{},
	}

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

	if w.LP_V3_Providers == nil {
		w.LP_V3_Providers = []*LP_V3{}
	}

	if w.LP_V3_Positions == nil {
		w.LP_V3_Positions = []*LP_V3_Position{}
	}

	if w.LP_V4_Providers == nil {
		w.LP_V4_Providers = []*LP_V4{}
	}

	if w.LP_V4_Positions == nil {
		w.LP_V4_Positions = []*LP_V4_Position{}
	}

	if w.Stakings == nil {
		w.Stakings = []*Staking{}
	}

	if w.StakingPositions == nil {
		w.StakingPositions = []*StakingPosition{}
	}

	if w.ParamInt == nil {
		w.ParamInt = make(map[string]int)
	}

	if w.ParamStr == nil {
		w.ParamStr = make(map[string]string)
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

func (w *Wallet) SetContract(a common.Address, c *Contract) error {
	w.writeMutex.Lock()
	defer w.writeMutex.Unlock()

	w.Contracts[a] = c

	return w._locked_Save()
}

func (w *Wallet) GetAddressByName(n string) *Address {
	for _, s := range w.Addresses {
		if s.Name == n {
			return s
		}
	}

	return w.GetAddress(n)
}

func (w *Wallet) GetToken(chain int, a string) *Token {
	if common.IsHexAddress(a) {
		t := w.GetTokenByAddress(chain, common.HexToAddress(a))
		if t != nil {
			return t
		}
	}

	return w.GetTokenBySymbol(chain, a)
}

func (w *Wallet) GetNativeToken(b *Blockchain) (*Token, error) {
	for _, t := range w.Tokens {
		if t.ChainId == b.ChainId && t.Native {
			return t, nil
		}
	}

	log.Error().Msgf("Native token not found for blockchain %s", b.Name)
	return nil, errors.New("native token not found")
}

func (w *Wallet) AddToken(chain int, a common.Address, n string, s string, d int) error {
	w.writeMutex.Lock()
	defer w.writeMutex.Unlock()

	if w.GetTokenByAddress(chain, a) != nil {
		return errors.New("token already exists")
	}

	t := &Token{
		ChainId:  chain,
		Address:  a,
		Name:     n,
		Symbol:   s,
		Decimals: d,
	}

	w.Tokens = append(w.Tokens, t)
	w._locked_MarkUniqueTokens()

	return w._locked_Save()
}

func (w *Wallet) GetTokenByAddress(chain int, a common.Address) *Token {
	if a.Cmp(common.Address{}) == 0 {
		b := w.GetBlockchain(chain)
		if b == nil {
			return nil
		}
		t, err := w.GetNativeToken(b)
		if err != nil {
			return nil
		}
		return t
	}

	for _, t := range w.Tokens {
		if t.ChainId == chain && (t.Address == a) {
			return t
		}
	}

	return nil
}

func (w *Wallet) GetTokenBySymbol(chain int, s string) *Token {
	for _, t := range w.Tokens {
		if t.ChainId == chain && t.Symbol == s {
			if !t.Unique {
				return nil // Ambiguous
			}
			return t
		}
	}
	return nil
}

func (w *Wallet) DeleteToken(chain int, a common.Address) error {
	w.writeMutex.Lock()
	defer w.writeMutex.Unlock()

	for i, t := range w.Tokens {
		if t.ChainId == chain && t.Address == a {
			w.Tokens = append(w.Tokens[:i], w.Tokens[i+1:]...)

			w._locked_MarkUniqueTokens()
			return w._locked_Save()
		}
	}

	return errors.New("token not found")
}

func (w *Wallet) _locked_MarkUniqueTokens() {
	for _, b := range w.Blockchains {
		for i, t := range w.Tokens {
			if t.ChainId == b.ChainId {
				t.Unique = true
				for j := 0; j < i; j++ {
					if w.Tokens[j].ChainId == b.ChainId && w.Tokens[j].Symbol == t.Symbol {
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

func (w *Wallet) AddBlockchain(b *Blockchain) error {
	w.writeMutex.Lock()
	defer w.writeMutex.Unlock()

	if w.GetBlockchain(b.ChainId) != nil {
		return errors.New("blockchain already exists")
	}

	if w.GetBlockchainByName(b.Name) != nil {
		return errors.New("blockchain with the same name already exists")
	}

	w.Blockchains = append(w.Blockchains, b)

	w._locked_AuditNativeTokens()

	return w._locked_Save()
}

func (w *Wallet) EditBlockchain(ub *Blockchain) error {
	w.writeMutex.Lock()
	defer w.writeMutex.Unlock()

	b := w.GetBlockchain(ub.ChainId)
	if b == nil {
		return errors.New("blockchain not found")
	}

	b.Name = ub.Name
	b.ShortName = ub.ShortName
	b.Url = ub.Url
	b.Currency = ub.Currency
	b.ExplorerUrl = ub.ExplorerUrl
	b.ExplorerAPIToken = ub.ExplorerAPIToken
	b.ExplorerAPIUrl = ub.ExplorerAPIUrl
	b.ExplorerApiType = ub.ExplorerApiType
	b.WTokenAddress = ub.WTokenAddress
	b.Multicall = ub.Multicall
	b.RPCRateLimit = ub.RPCRateLimit
	b.RPCRateAuto = ub.RPCRateAuto

	w._locked_AuditNativeTokens()

	return w._locked_Save()

}

// LP V2 methods

func (w *Wallet) AddLP_V2(lp *LP_V2) error {
	if w.GetLP_V2(lp.ChainId, lp.Factory) != nil {
		return errors.New("provider already exists")
	}

	if w.GetLP_V2_by_name(lp.ChainId, lp.Name) != nil {
		return errors.New("provider with the same name already exists")
	}

	w.LP_V2_Providers = append(w.LP_V2_Providers, lp)
	return w.Save()
}

func (w *Wallet) GetLP_V2(chainId int, factory common.Address) *LP_V2 {
	for _, lp := range w.LP_V2_Providers {
		if lp.ChainId == chainId && lp.Factory == factory {
			return lp
		}
	}
	return nil
}

func (w *Wallet) GetLP_V2_by_name(chainId int, name string) *LP_V2 {
	for _, lp := range w.LP_V2_Providers {
		if lp.ChainId == chainId && lp.Name == name {
			return lp
		}
	}
	return nil
}

func (w *Wallet) RemoveLP_V2(chainId int, factory common.Address) error {
	w.writeMutex.Lock()
	defer w.writeMutex.Unlock()

	for i, lp := range w.LP_V2_Providers {
		if lp.ChainId == chainId && lp.Factory == factory {
			w.LP_V2_Providers = append(w.LP_V2_Providers[:i], w.LP_V2_Providers[i+1:]...)

			// remove all positions
			for j := len(w.LP_V2_Positions) - 1; j >= 0; j-- {
				if w.LP_V2_Positions[j].ChainId == chainId && w.LP_V2_Positions[j].Factory == factory {
					w.LP_V2_Positions = append(w.LP_V2_Positions[:j], w.LP_V2_Positions[j+1:]...)
				}
			}

			return w._locked_Save()
		}
	}

	return errors.New("provider not found")
}

func (w *Wallet) AddLP_V2Position(lp *LP_V2_Position) error {
	w.writeMutex.Lock()
	defer w.writeMutex.Unlock()

	if pos := w.GetLP_V2Position(lp.ChainId, lp.Factory, lp.Pair); pos != nil {
		// update
		pos.Owner = lp.Owner
		pos.Token0 = lp.Token0
		pos.Token1 = lp.Token1
	} else {
		w.LP_V2_Positions = append(w.LP_V2_Positions, lp)
	}
	return w._locked_Save()
}

func (w *Wallet) GetLP_V2Position(chainId int, factory common.Address, pair common.Address) *LP_V2_Position {
	for _, lp := range w.LP_V2_Positions {
		if lp.ChainId == chainId &&
			lp.Factory == factory &&
			lp.Pair == pair {
			return lp
		}
	}
	return nil
}

func (w *Wallet) RemoveLP_V2Position(addr common.Address, chainId int, factory common.Address, pair common.Address) error {
	w.writeMutex.Lock()
	defer w.writeMutex.Unlock()

	for i, lp := range w.LP_V2_Positions {
		if lp.Owner == addr && lp.ChainId == chainId && lp.Factory == factory && lp.Pair == pair {
			w.LP_V2_Positions = append(w.LP_V2_Positions[:i], w.LP_V2_Positions[i+1:]...)
			return w._locked_Save()
		}
	}
	return errors.New("position not found")
}

// LP V3 methods

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
	w.writeMutex.Lock()
	defer w.writeMutex.Unlock()

	for i, lp := range w.LP_V3_Providers {
		if lp.ChainId == chainId && lp.Provider == provider {
			w.LP_V3_Providers = append(w.LP_V3_Providers[:i], w.LP_V3_Providers[i+1:]...)

			// remove all positions
			for j := len(w.LP_V3_Positions) - 1; j >= 0; j-- {
				if w.LP_V3_Positions[j].ChainId == chainId && w.LP_V3_Positions[j].Provider == provider {
					w.LP_V3_Positions = append(w.LP_V3_Positions[:j], w.LP_V3_Positions[j+1:]...)
				}
			}

			return w._locked_Save()
		}
	}

	return errors.New("provider not found")
}

func (w *Wallet) AddLP_V3Position(lp *LP_V3_Position) error {
	w.writeMutex.Lock()
	defer w.writeMutex.Unlock()

	if pos := w.GetLP_V3Position(lp.ChainId, lp.Provider, lp.NFT_Token); pos != nil {
		// update
		pos.Owner = lp.Owner
		pos.Token0 = lp.Token0
		pos.Token1 = lp.Token1
		pos.Pool = lp.Pool
		pos.Fee = lp.Fee
	} else {
		w.LP_V3_Positions = append(w.LP_V3_Positions, lp)
	}
	return w._locked_Save()
}

func (w *Wallet) GetLP_V3Position(chainId int, provider common.Address, nft *big.Int) *LP_V3_Position {
	for _, lp := range w.LP_V3_Positions {
		if lp.ChainId == chainId &&
			lp.Provider.Cmp(provider) == 0 &&
			lp.NFT_Token.Cmp(nft) == 0 {
			return lp
		}
	}
	return nil
}

func (w *Wallet) RemoveLP_V3Position(addr common.Address, chainId int, provider common.Address, nft *big.Int) error {
	w.writeMutex.Lock()
	defer w.writeMutex.Unlock()

	for i, lp := range w.LP_V3_Positions {
		if lp.Owner.Cmp(addr) == 0 && lp.ChainId == chainId && lp.Provider == provider && lp.NFT_Token.Cmp(nft) == 0 {
			w.LP_V3_Positions = append(w.LP_V3_Positions[:i], w.LP_V3_Positions[i+1:]...)
			return w._locked_Save()
		}
	}
	return errors.New("position not found")
}

// LP V4 methods

func (w *Wallet) AddLP_V4(lp *LP_V4) error {
	if w.GetLP_V4(lp.ChainId, lp.Provider) != nil {
		return errors.New("provider already exists")
	}

	if w.GetLP_V4_by_name(lp.ChainId, lp.Name) != nil {
		return errors.New("provider with the same name already exists")
	}

	w.LP_V4_Providers = append(w.LP_V4_Providers, lp)
	return w.Save()
}

func (w *Wallet) GetLP_V4(chainId int, addr common.Address) *LP_V4 {
	for _, lp := range w.LP_V4_Providers {
		if lp.ChainId == chainId && lp.Provider == addr {
			return lp
		}
	}
	return nil
}

func (w *Wallet) GetLP_V4_by_name(chainId int, name string) *LP_V4 {
	for _, lp := range w.LP_V4_Providers {
		if lp.ChainId == chainId && lp.Name == name {
			return lp
		}
	}
	return nil
}

func (w *Wallet) RemoveLP_V4(chainId int, provider common.Address) error {
	w.writeMutex.Lock()
	defer w.writeMutex.Unlock()

	for i, lp := range w.LP_V4_Providers {
		if lp.ChainId == chainId && lp.Provider == provider {
			w.LP_V4_Providers = append(w.LP_V4_Providers[:i], w.LP_V4_Providers[i+1:]...)

			// remove all positions
			for j := len(w.LP_V4_Positions) - 1; j >= 0; j-- {
				if w.LP_V4_Positions[j].ChainId == chainId && w.LP_V4_Positions[j].Provider == provider {
					w.LP_V4_Positions = append(w.LP_V4_Positions[:j], w.LP_V4_Positions[j+1:]...)
				}
			}

			return w._locked_Save()
		}
	}

	return errors.New("provider not found")
}

func (w *Wallet) AddLP_V4Position(lp *LP_V4_Position) error {
	w.writeMutex.Lock()
	defer w.writeMutex.Unlock()

	if pos := w.GetLP_V4Position(lp.ChainId, lp.Provider, lp.NFT_Token); pos != nil {
		// update
		pos.Owner = lp.Owner
		pos.Currency0 = lp.Currency0
		pos.Currency1 = lp.Currency1
		pos.PoolId = lp.PoolId
		pos.Fee = lp.Fee
		pos.TickLower = lp.TickLower
		pos.TickUpper = lp.TickUpper
		pos.Liquidity = lp.Liquidity
		pos.HookAddress = lp.HookAddress
	} else {
		w.LP_V4_Positions = append(w.LP_V4_Positions, lp)
	}
	return w._locked_Save()
}

func (w *Wallet) GetLP_V4Position(chainId int, provider common.Address, nft *big.Int) *LP_V4_Position {
	for _, lp := range w.LP_V4_Positions {
		if lp.ChainId == chainId &&
			lp.Provider.Cmp(provider) == 0 &&
			lp.NFT_Token.Cmp(nft) == 0 {
			return lp
		}
	}
	return nil
}

func (w *Wallet) RemoveLP_V4Position(addr common.Address, chainId int, provider common.Address, nft *big.Int) error {
	w.writeMutex.Lock()
	defer w.writeMutex.Unlock()

	for i, lp := range w.LP_V4_Positions {
		if lp.Owner.Cmp(addr) == 0 && lp.ChainId == chainId && lp.Provider == provider && lp.NFT_Token.Cmp(nft) == 0 {
			w.LP_V4_Positions = append(w.LP_V4_Positions[:i], w.LP_V4_Positions[i+1:]...)
			return w._locked_Save()
		}
	}
	return errors.New("position not found")
}

func (w *Wallet) SetParamInt(name string, val int) error {
	w.writeMutex.Lock()
	defer w.writeMutex.Unlock()

	w.ParamInt[name] = val
	return w._locked_Save()
}

func (w *Wallet) SetParamStr(name string, val string) error {
	w.writeMutex.Lock()
	defer w.writeMutex.Unlock()

	w.ParamStr[name] = val
	return w._locked_Save()
}

// Staking methods

func (w *Wallet) AddStaking(s *Staking) error {
	w.writeMutex.Lock()
	defer w.writeMutex.Unlock()

	if w.GetStaking(s.ChainId, s.Contract) != nil {
		return errors.New("staking already exists")
	}

	if w.GetStakingByName(s.ChainId, s.Name) != nil {
		return errors.New("staking with the same name already exists")
	}

	w.Stakings = append(w.Stakings, s)
	return w._locked_Save()
}

func (w *Wallet) GetStaking(chainId int, contract common.Address) *Staking {
	for _, s := range w.Stakings {
		if s.ChainId == chainId && s.Contract == contract {
			return s
		}
	}
	return nil
}

func (w *Wallet) GetStakingByName(chainId int, name string) *Staking {
	for _, s := range w.Stakings {
		if s.ChainId == chainId && s.Name == name {
			return s
		}
	}
	return nil
}

func (w *Wallet) RemoveStaking(chainId int, contract common.Address) error {
	w.writeMutex.Lock()
	defer w.writeMutex.Unlock()

	for i, s := range w.Stakings {
		if s.ChainId == chainId && s.Contract == contract {
			w.Stakings = append(w.Stakings[:i], w.Stakings[i+1:]...)

			// remove all positions for this staking contract
			for j := len(w.StakingPositions) - 1; j >= 0; j-- {
				if w.StakingPositions[j].ChainId == chainId && w.StakingPositions[j].Contract == contract {
					w.StakingPositions = append(w.StakingPositions[:j], w.StakingPositions[j+1:]...)
				}
			}

			return w._locked_Save()
		}
	}

	return errors.New("staking not found")
}

func (w *Wallet) AddStakingPosition(pos *StakingPosition) error {
	w.writeMutex.Lock()
	defer w.writeMutex.Unlock()

	// For validator-based staking, check with ValidatorId
	if pos.ValidatorId > 0 {
		if existing := w.GetStakingPositionWithValidator(pos.ChainId, pos.Contract, pos.Owner, pos.ValidatorId); existing != nil {
			return nil
		}
	} else {
		if existing := w.GetStakingPosition(pos.ChainId, pos.Contract, pos.Owner); existing != nil {
			return nil
		}
	}

	w.StakingPositions = append(w.StakingPositions, pos)
	return w._locked_Save()
}

func (w *Wallet) GetStakingPosition(chainId int, contract common.Address, owner common.Address) *StakingPosition {
	for _, p := range w.StakingPositions {
		if p.ChainId == chainId && p.Contract == contract && p.Owner == owner {
			return p
		}
	}
	return nil
}

func (w *Wallet) GetStakingPositionWithValidator(chainId int, contract common.Address, owner common.Address, validatorId uint64) *StakingPosition {
	for _, p := range w.StakingPositions {
		if p.ChainId == chainId && p.Contract == contract && p.Owner == owner && p.ValidatorId == validatorId {
			return p
		}
	}
	return nil
}

func (w *Wallet) RemoveStakingPosition(chainId int, contract common.Address, owner common.Address) error {
	w.writeMutex.Lock()
	defer w.writeMutex.Unlock()

	for i, p := range w.StakingPositions {
		if p.ChainId == chainId && p.Contract == contract && p.Owner == owner {
			w.StakingPositions = append(w.StakingPositions[:i], w.StakingPositions[i+1:]...)
			return w._locked_Save()
		}
	}
	return errors.New("staking position not found")
}

func (w *Wallet) RemoveStakingPositionWithValidator(chainId int, contract common.Address, owner common.Address, validatorId uint64) error {
	w.writeMutex.Lock()
	defer w.writeMutex.Unlock()

	for i, p := range w.StakingPositions {
		if p.ChainId == chainId && p.Contract == contract && p.Owner == owner && p.ValidatorId == validatorId {
			w.StakingPositions = append(w.StakingPositions[:i], w.StakingPositions[i+1:]...)
			return w._locked_Save()
		}
	}
	return errors.New("staking position not found")
}
