package main

import (
	"github.com/keithzetterstrom/BibCoin/cmd/api"
	"github.com/keithzetterstrom/BibCoin/internal/pkg/blockchain"
	clipkg "github.com/keithzetterstrom/BibCoin/tools/cli"
)

func main() {
	bc, err := blockchain.NewBlockchain()
	if err != nil {
		bc = blockchain.CreateBlockchain("maxa")
	}
	defer bc.Db.Close()

	cli := clipkg.NewFlagCLI()

	router := api.NewRouter(bc, cli)
	router.Start()
}
