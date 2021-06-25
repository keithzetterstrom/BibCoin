package wallet

import (
	"bytes"
	"crypto/elliptic"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/keithzetterstrom/BibCoin/tools/base58"
	"io/ioutil"
	"log"
	"os"
)

type Wallets struct {
	Wallets    map[string]*Wallet
	filePath   string
	walletPath string
}

type Address struct {
	Address string `json:"address"`
}

// NewWallets returns wallets from file
func NewWallets(fileAddr, fileWallet string) (*Wallets, error) {
	wallets := Wallets{filePath: fileAddr, walletPath: fileWallet}
	wallets.Wallets = make(map[string]*Wallet)

	err := wallets.LoadFromFile()

	return &wallets, err
}

// CreateWallet creates new wallet and returns its address
func (ws *Wallets) CreateWallet() string {
	wallet := NewWallet()
	address := fmt.Sprintf("%s", wallet.GetAddress())

	addr := Address{Address: address}
	addrByte, _ := json.Marshal(addr)

	err := ioutil.WriteFile(ws.filePath, addrByte, 0664)
	if err != nil {
		log.Panic(err)
	}

	ws.Wallets[address] = wallet

	return address
}

// GetAddresses returns all addresses
func (ws *Wallets) GetAddresses() []string {
	var addresses []string

	for address := range ws.Wallets {
		addresses = append(addresses, address)
	}

	return addresses
}

// GetWallet returns Wallet by address
func (ws Wallets) GetWallet(address string) (Wallet, error) {
	if _, ok := ws.Wallets[address]; !ok {
		return Wallet{}, errors.New("Wallet permissions denied ")
	}
	return *ws.Wallets[address], nil
}

// LoadFromFile gets Wallets from file
func (ws *Wallets) LoadFromFile() error {
	if _, err := os.Stat(ws.walletPath); os.IsNotExist(err) {
		return err
	}

	fileContent, err := ioutil.ReadFile(ws.walletPath)
	if err != nil {
		log.Panic(err)
	}

	var wallets Wallets
	gob.Register(elliptic.P256())
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decoder.Decode(&wallets)
	if err != nil {
		log.Panic(err)
	}

	ws.Wallets = wallets.Wallets

	return nil
}

// SaveToFile saves Wallets to file
func (ws Wallets) SaveToFile() {
	var content bytes.Buffer

	gob.Register(elliptic.P256())

	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(ws)
	if err != nil {
		log.Panic(err)
	}

	err = ioutil.WriteFile(ws.walletPath, content.Bytes(), 0644)
	if err != nil {
		log.Panic(err)
	}
}

// ValidateAddress returns true if address is valid
func ValidateAddress(address string) bool {
	pubKeyHash := base58.DecodeBase58([]byte(address))
	actualChecksum := pubKeyHash[len(pubKeyHash) - addressChecksumLen:]
	version := pubKeyHash[0]
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash) - addressChecksumLen]
	targetChecksum := checksum(append([]byte{version}, pubKeyHash...))

	return bytes.Compare(actualChecksum, targetChecksum) == 0
}

func (ws Wallets) PrintWallets() {
	fmt.Println("--------------- List of wallets addresses ---------------")
	for k := range ws.Wallets {
		fmt.Printf("\t%s\n", k)
	}
	fmt.Println("---------------------------------------------------------")
}
