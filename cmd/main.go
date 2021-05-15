package main

import (
	"fmt"
	"github.com/keithzetterstrom/BibCoin/cmd/api"
	"github.com/keithzetterstrom/BibCoin/internal/pkg/blockchain"
	"github.com/keithzetterstrom/BibCoin/internal/pkg/wallet"
	clipkg "github.com/keithzetterstrom/BibCoin/tools/cli"
)

func main() {
	wallets, err := wallet.NewWallets()

	bc, err := blockchain.NewBlockchain()
	if err != nil {
		addr := wallets.CreateWallet()
		wallets.SaveToFile()
		bc = blockchain.CreateBlockchain(addr)
		fmt.Println(addr)
		return
	}
	defer bc.Db.Close()

	cli := clipkg.NewFlagCLI()

	router := api.NewRouter(bc, cli, wallets)
	router.Start()
}
