package main

import (
	"encoding/json"
	"fmt"
	"github.com/keithzetterstrom/BibCoin/internal/pkg/blockchain"
	"github.com/keithzetterstrom/BibCoin/internal/pkg/network"
	"github.com/keithzetterstrom/BibCoin/internal/pkg/wallet"
	"golang.org/x/mobile/app"
	"io/ioutil"
)

const fullNodeAddress = "192.168.1.64:9000"
const addrFile = "/sdcard/addr.json"
const dbFile = "/sdcard/Blockchain.db"
const walletFile = "/sdcard/wallet.dat"

func main() {
	app.Main(func(a app.App) {
		wallets, err := wallet.NewWallets(addrFile, walletFile)

		bc, err := blockchain.NewBlockchain(dbFile, addrFile, walletFile)
		if err != nil {
			addr := wallets.CreateWallet()
			wallets.SaveToFile()
			bc = blockchain.CreateEmptyBlockchain(dbFile, addrFile, walletFile)

			// for full node
			bc.AddGenesisBlock(addr)
		}
		defer bc.Db.Close()

		addrByte, _ := ioutil.ReadFile(addrFile)
		addr := &wallet.Address{}
		_ = json.Unmarshal(addrByte, addr)

		fmt.Println("Your address:", addr.Address)
		nw := network.NewNetwork(bc, fullNodeAddress, addr.Address)

		nw.StartFullServer()
	})
}
