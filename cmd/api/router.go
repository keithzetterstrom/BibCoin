package api

import (
	"fmt"
	blockchainpkg "github.com/keithzetterstrom/BibCoin/internal/pkg/blockchain"
	clipkg "github.com/keithzetterstrom/BibCoin/tools/cli"
	"os"
	"strconv"
)

type router struct {
	blockchain *blockchainpkg.Blockchain
	cli        *clipkg.FlagsCLI
}

type Router interface {
	Start()
}

func NewRouter(blockchain *blockchainpkg.Blockchain, cli *clipkg.FlagsCLI) Router {
	return &router{
		blockchain: blockchain,
		cli: cli,
	}
}

func (r * router) Start() {
	r.cli.FlagsCLI()

	switch {
	case r.cli.SendCmd != "" && os.Args[3] != "" && os.Args[4] != "":
		coins, _ := strconv.Atoi(os.Args[4])
		r.send(r.cli.SendCmd, os.Args[3], coins)

	case r.cli.PrintChainCmd:
		r.printChain()

	case r.cli.BalanceCmd != "":
		r.getBalance(r.cli.BalanceCmd)

	default:
		r.cli.PrintUsage()
	}
}

func (r * router) getBalance(address string) {
	balance := 0
	unspentTxOutputs := r.blockchain.FindUnspentTxOutputs(address)

	for _, out := range unspentTxOutputs {
		balance += out.Value
	}

	fmt.Printf("Balance of '%s': %d\n", address, balance)
}

func (r * router) send(from, to string, amount int) {
	tx := blockchainpkg.NewTransaction(from, to, amount, r.blockchain)
	r.blockchain.MineBlock([]*blockchainpkg.Transaction{tx})
	fmt.Println("Success!")
}

func (r * router) printChain() {
	iterator := r.blockchain.NewIterator()
	for {
		block := iterator.Next()

		fmt.Printf("Prev. hash: %x\n", block.PrevBlockHash)
		fmt.Printf("Data: %v\n", block.Transactions)
		fmt.Printf("Hash: %x\n", block.Hash)

		pow := blockchainpkg.NewProofOfWork(block)

		fmt.Printf("PoW: %s\n", strconv.FormatBool(pow.Validate()))
		fmt.Println()

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
}
