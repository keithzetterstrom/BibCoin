package main

import (
	"encoding/json"
	"fmt"
	"github.com/keithzetterstrom/BibCoin/cmd/api"
	"github.com/keithzetterstrom/BibCoin/internal/pkg/blockchain"
	"github.com/keithzetterstrom/BibCoin/internal/pkg/network"
	"github.com/keithzetterstrom/BibCoin/internal/pkg/wallet"
	clipkg "github.com/keithzetterstrom/BibCoin/tools/cli"
	"io/ioutil"
)

const nodeAddress = "172.20.10.9:9001"
const addrFile = "addr.json"
const dbFile = "Blockchain.db"
const walletFile = "wallet.dat"

func main() {
	wallets, err := wallet.NewWallets(addrFile, walletFile)

	bc, err := blockchain.NewBlockchain(dbFile, addrFile, walletFile)
	if err != nil {
		addr := wallets.CreateWallet()
		wallets.SaveToFile()
		bc = blockchain.CreateEmptyBlockchain(dbFile, addrFile, walletFile)
		fmt.Println("Your address:", addr)
	}
	defer bc.Db.Close()

	addrByte, _ := ioutil.ReadFile(addrFile)
	addr := &wallet.Address{}
	_ = json.Unmarshal(addrByte, addr)

	nw := network.NewNetwork(bc, nodeAddress, addr.Address)

	cli := clipkg.NewFlagCLI()

	router := api.NewRouter(bc, cli, wallets, nw)
	router.Start()
}
