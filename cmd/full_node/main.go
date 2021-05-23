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

const fullNodeAddress = "127.0.0.1:9000"
const nodeID = "0"

func main() {
	wallets, err := wallet.NewWallets()

	bc, err := blockchain.NewBlockchain(nodeID)
	if err != nil {
		addr := wallets.CreateWallet()
		wallets.SaveToFile()
		bc = blockchain.CreateEmptyBlockchain(nodeID)
		// for full node
		bc.AddGenesisBlock(addr)
		fmt.Println(addr)
	}
	defer bc.Db.Close()

	addrByte, _ := ioutil.ReadFile("addr.json")
	addr := &wallet.Address{}
	_ = json.Unmarshal(addrByte, addr)

	nw := network.NewNetwork(bc, fullNodeAddress, addr.Address)

	cli := clipkg.NewFlagCLI()

	router := api.NewRouter(bc, cli, wallets, nw)

	router.Start()
}
