package wallet

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"io"
	"os"
	"sort"

	"github.com/AlexNa-Holdings/web3pro/address"
	"github.com/AlexNa-Holdings/web3pro/blockchain"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/signer"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/pbkdf2"
)

const SOLT_SIZE = 32

type Wallet struct {
	Name        string                   `json:"name"`
	Blockchains []*blockchain.Blockchain `json:"blockchains"`
	Signers     []*signer.Signer         `json:"signers"`
	Addresses   []*address.Address       `json:"addresses"`
	FilePath    string                   `json:"-"`
	Password    string                   `json:"-"`
}

var CurrentWallet *Wallet

func Open(name string, pass string) error {

	w, err := OpenFromFile(cmn.DataFolder+"/wallets/"+name, pass)

	if err == nil {
		CurrentWallet = w
		CurrentWallet.FilePath = cmn.DataFolder + "/wallets/" + name
		CurrentWallet.Password = pass
	}
	return err
}

func (w *Wallet) Save() error {
	return SaveToFile(w, w.FilePath, w.Password)
}

func Exists(name string) bool {
	_, err := os.Stat(cmn.DataFolder + "/wallets/" + name)
	return !os.IsNotExist(err)
}

func Create(name, pass string) error {
	w := &Wallet{}

	return SaveToFile(w, cmn.DataFolder+"/wallets/"+name, pass)
}

func List() []string {
	files, err := os.ReadDir(cmn.DataFolder + "/wallets")
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

func OpenFromFile(file string, pass string) (*Wallet, error) {
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

func (w *Wallet) GetSigner(n string) *signer.Signer {
	for _, s := range w.Signers {
		if s.Name == n {
			return s
		}
	}
	return nil
}

func (w *Wallet) GetAddress(a string) *address.Address {
	for _, s := range w.Addresses {
		if s.Address.String() == a {
			return s
		}
	}
	return nil
}

func (w *Wallet) GetAddressByName(n string) *address.Address {
	for _, s := range w.Addresses {
		if s.Name == n {
			return s
		}
	}
	return nil
}

type ByNameAndCopy []*signer.Signer

func (a ByNameAndCopy) Len() int      { return len(a) }
func (a ByNameAndCopy) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByNameAndCopy) Less(i, j int) bool {

	ni := a[i].Name
	if a[i].CopyOf != "" {
		ni = a[i].CopyOf
	}

	nj := a[j].Name
	if a[j].CopyOf != "" {
		nj = a[j].CopyOf
	}

	if ni == nj {
		return a[i].CopyOf < a[j].CopyOf
	} else {
		return ni < nj
	}
}

func (w *Wallet) SortSigners() {
	sort.Sort(ByNameAndCopy(w.Signers))
}
