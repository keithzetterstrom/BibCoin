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
		fmt.Println("last index: ", lastIndex)
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
		}
	} else {
		r.network.SendTx(r.network.KnownNodes[0], tx)
		fmt.Println("VerifyTransaction: ", r.blockchain.VerifyTransaction(tx))
	}

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

func (r * router) startNode()  {
	r.network.StartServer()
}

func (r * router) startMiningNode()  {
	r.network.StartMineServer()
}

func (r * router) startFullNode()  {
	r.network.StartFullServer()
}
