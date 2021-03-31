package main

import (
	"github.com/keithzetterstrom/BibCoin/cmd/api"
	"github.com/keithzetterstrom/BibCoin/internal/pkg/blockchain"
	clipkg "github.com/keithzetterstrom/BibCoin/tools/cli"
	iteratorpkg "github.com/keithzetterstrom/BibCoin/tools/iterator"
)

func main() {
	bc := blockchain.NewBlockchain()
	defer bc.Db.Close()

	cli := clipkg.NewFlagCLI()
	iterator := iteratorpkg.NewIterator(bc)

	router := api.NewRouter(bc, cli, iterator)
	router.Start()
}
