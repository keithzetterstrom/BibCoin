package api

import (
	"fmt"
	blockchainpkg "github.com/keithzetterstrom/BibCoin/internal/pkg/blockchain"
	clipkg "github.com/keithzetterstrom/BibCoin/tools/cli"
	iteratorpkg "github.com/keithzetterstrom/BibCoin/tools/iterator"
	"strconv"
)

type router struct {
	blockchain *blockchainpkg.Blockchain
	cli        *clipkg.FlagsCLI
	iterator   *iteratorpkg.Iterator
}

type Router interface {
	Start()
}

func NewRouter(blockchain *blockchainpkg.Blockchain, cli *clipkg.FlagsCLI, iterator *iteratorpkg.Iterator) Router {
	return &router{
		blockchain: blockchain,
		cli: cli,
		iterator: iterator,
	}
}

func (r * router) Start() {
	r.cli.FlagsCLI()

	switch {
	case r.cli.AddBlockCmd != "":
		r.blockchain.AddBlock(r.cli.AddBlockCmd)

	case r.cli.PrintChainCmd:
		r.printChain()

	default:
		r.cli.PrintUsage()
	}
}

func (r * router) printChain() {
	for {
		block := r.iterator.Next()

		fmt.Printf("Prev. hash: %x\n", block.PrevBlockHash)
		fmt.Printf("Data: %s\n", block.Data)
		fmt.Printf("Hash: %x\n", block.Hash)

		pow := blockchainpkg.NewProofOfWork(block)

		fmt.Printf("PoW: %s\n", strconv.FormatBool(pow.Validate()))
		fmt.Println()

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
}
