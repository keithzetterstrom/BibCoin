package api

import (
	"fmt"
	blockchainpkg "github.com/keithzetterstrom/BibCoin/internal/pkg/blockchain"
	walletpkg "github.com/keithzetterstrom/BibCoin/internal/pkg/wallet"
	"github.com/keithzetterstrom/BibCoin/tools/base58"
	clipkg "github.com/keithzetterstrom/BibCoin/tools/cli"
	"log"
	"os"
	"strconv"
)

type router struct {
	blockchain *blockchainpkg.Blockchain
	cli        *clipkg.FlagsCLI
	wallets    *walletpkg.Wallets
}

type Router interface {
	Start()
}

func NewRouter(blockchain *blockchainpkg.Blockchain, cli *clipkg.FlagsCLI, wallets *walletpkg.Wallets) Router {
	return &router{
		blockchain: blockchain,
		cli: cli,
		wallets: wallets,
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

	case r.cli.CreateWallet:
		r.createWallet()

	case r.cli.ShowWallets:
		r.showWallets()

	default:
		r.cli.PrintUsage()
	}
}

func (r * router) getBalance(address string) {
	if !walletpkg.ValidateAddress(address) {
		log.Panic("invalid address")
	}

	balance := 0

	pubKeyHash := base58.DecodeBase58([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash) - 4]

	unspentTxOutputs := r.blockchain.FindUnspentTxOutputs(pubKeyHash)

	for _, out := range unspentTxOutputs {
		balance += out.Value
	}

	fmt.Printf("Balance of '%s': %d\n", address, balance)
}

func (r * router) send(from, to string, amount int) {
	if !walletpkg.ValidateAddress(from) {
		log.Panic("invalid address")
	}

	if !walletpkg.ValidateAddress(to) {
		log.Panic("invalid address")
	}

	tx := blockchainpkg.NewTransaction(from, to, amount, r.blockchain)
	r.blockchain.MineBlock([]*blockchainpkg.Transaction{tx})
	fmt.Println("Success!")
}

func (r * router) printChain() {
	iterator := r.blockchain.NewIterator()
	fmt.Println("-------------------------------- BlockChain --------------------------------")
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
	fmt.Println("---------------------------------- * * * ----------------------------------")
}

func (r * router) createWallet()  {
	fmt.Println("New address: ", r.wallets.CreateWallet())
	r.wallets.SaveToFile()
}

func (r * router) showWallets()  {
	r.wallets.PrintWallets()
}
