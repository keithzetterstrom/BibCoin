package api

import (
	"fmt"
	blockchainpkg "github.com/keithzetterstrom/BibCoin/internal/pkg/blockchain"
	networkpkg "github.com/keithzetterstrom/BibCoin/internal/pkg/network"
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
	network    *networkpkg.Network
}

type Router interface {
	Start()
}

// NewRouter returns new Router
func NewRouter(
	blockchain *blockchainpkg.Blockchain,
	cli *clipkg.FlagsCLI,
	wallets *walletpkg.Wallets,
	network *networkpkg.Network,
) Router {
	return &router{
		blockchain: blockchain,
		cli: cli,
		wallets: wallets,
		network: network,
	}
}

// Start starts cli
func (r * router) Start() {
	r.cli.FlagsCLI()

	switch {
	case r.cli.SendCmd != "" && os.Args[3] != "" && os.Args[4] != "":
		coins, _ := strconv.Atoi(os.Args[4])
		r.send(r.cli.SendCmd, os.Args[3], coins, false)

	case r.cli.PrintChainCmd:
		r.printChain()

	case r.cli.BalanceCmd != "":
		r.getBalance(r.cli.BalanceCmd)

	case r.cli.CreateWallet:
		r.createWallet()

	case r.cli.ShowWallets:
		r.showWallets()

	case r.cli.StartNode:
		r.startNode()

	case r.cli.StartMiningNode:
		r.startMiningNode()

	case r.cli.StartFullNode:
		r.startFullNode()

	default:
		r.cli.PrintUsage()
	}
}

// getBalance returns balance of the given address
func (r * router) getBalance(address string) {
	if !walletpkg.ValidateAddress(address) {
		log.Panic("invalid address")
	}

	balance := 0

	pubKeyHash := base58.DecodeBase58([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash) - 4]

	unspentTxOutputs := r.blockchain.FindUnspentTxOutputs(pubKeyHash)

	for _, out := range unspentTxOutputs {
		balance += len(out.Value)
	}

	fmt.Printf("Balance of '%s': %d\n", address, balance)
}

// send sends coins from one to another address
//
// If mineNow - true: node appends new block with new transaction locally
// bypassing mining stage and follow the satoshi stage
func (r * router) send(from, to string, amount int, mineNow bool) {
	if !walletpkg.ValidateAddress(from) {
		fmt.Println("Invalid address")
		return
	}

	if !walletpkg.ValidateAddress(to) {
		fmt.Println("Invalid address")
		return
	}

	tx, err := blockchainpkg.NewTransaction(from, to, amount, r.blockchain)
	if err != nil {
		fmt.Println("Failed:", err)
		return
	}
	if mineNow {
		lastIndex, err := r.blockchain.GetLastSatoshiIndex()
		if err != nil {
			fmt.Println("Failed:", err)
			return
		}
		cbTx := blockchainpkg.NewCoinbaseTX(from, from, "", lastIndex)
		txs := []*blockchainpkg.Transaction{cbTx, tx}

		block := r.blockchain.MineBlock(from)
		_, err = r.blockchain.AddNewBlock(block, txs, from)
		if err != nil {
			fmt.Println(err)
			return
		}
	} else {
		r.network.SendTx(r.network.KnownNodes[0], tx)
	}

	fmt.Println("Success!")
}

// printChain prints blocks with their hash and previous hash
func (r * router) printChain() {
	iterator := r.blockchain.NewIterator()
	fmt.Println("-------------------------------- BlockChain --------------------------------")
	for {
		block := iterator.Next()

		fmt.Printf("Prev. hash: %x\n", block.PrevBlockHash)
		fmt.Printf("Hash: %x\n", block.Hash)

		fmt.Println()

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
	fmt.Println("---------------------------------- * * * ----------------------------------")
}

// createWallet creates Wallet and prints address
func (r * router) createWallet()  {
	fmt.Println("New address: ", r.wallets.CreateWallet())
	r.wallets.SaveToFile()
}

// showWallets prints wallet's addresses
func (r * router) showWallets()  {
	r.wallets.PrintWallets()
}

// startNode starts synchronization
func (r * router) startNode()  {
	r.network.StartServer()
}

// startMiningNode starts miner node
func (r * router) startMiningNode()  {
	r.network.StartMineServer()
}

// startFullNode starts full node
func (r * router) startFullNode()  {
	r.network.StartFullServer()
}
