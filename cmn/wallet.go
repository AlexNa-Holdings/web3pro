package cmn

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"io"
	"os"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/pbkdf2"
)

const SOLT_SIZE = 32

var CurrentWallet *Wallet

func Open(name string, pass string) error {

	w, err := openFromFile(DataFolder+"/wallets/"+name, pass)

	if err == nil {
		w.FilePath = DataFolder + "/wallets/" + name
		w.Password = pass
		w.WriteMutex = sync.Mutex{}

		w.AuditNativeTokens()
		w.MarkUniqueTokens()
		CurrentWallet = w
	}
	return err
}

func (w *Wallet) AuditNativeTokens() {

	eddited := false

	for _, b := range w.Blockchains {
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
	return nil
}

func (w *Wallet) Save() error {
	return SaveToFile(w, w.FilePath, w.Password)
}

func Exists(name string) bool {
	_, err := os.Stat(DataFolder + "/wallets/" + name)
	return !os.IsNotExist(err)
}

func Create(name, pass string) error {
	w := &Wallet{}

	return SaveToFile(w, DataFolder+"/wallets/"+name, pass)
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
	w.WriteMutex.Lock()
	defer w.WriteMutex.Unlock()

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

func (w *Wallet) GetSignerWithCopy(name string) (*Signer, int) {
	for _, s := range w.Signers {
		for j, c := range s.Copies {
			if c == name {
				return s, j
			}
		}
	}
	return nil, -1
}

func (w *Wallet) GetAddress(a string) *Address {
	for _, s := range w.Addresses {
		if s.Address.String() == a {
			return s
		}
	}
	return nil
}

func (w *Wallet) GetAddressByName(n string) *Address {
	for _, s := range w.Addresses {
		if s.Name == n {
			return s
		}
	}
	return nil
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

func (w *Wallet) GetTokenByAddress(b string, a common.Address) *Token {
	for _, t := range w.Tokens {
		if t.Blockchain == b && ((!t.Native && t.Address == a) ||
			(t.Native && a == common.Address{})) {
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
