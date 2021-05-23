package cli

import (
	"flag"
	"fmt"
)

type FlagsCLI struct {
	SendCmd         string
	PrintChainCmd   bool
	BalanceCmd      string
	CreateWallet    bool
	ShowWallets     bool
	StartNode       bool
	StartFullNode   bool
	StartMiningNode bool
}

func NewFlagCLI() *FlagsCLI {
	return &FlagsCLI{}
}

func (f *FlagsCLI) FlagsCLI()  {
	flag.StringVar(&f.SendCmd, "s", "", "")
	flag.BoolVar(&f.PrintChainCmd, "p", false, "")
	flag.StringVar(&f.BalanceCmd, "b", "", "")
	flag.BoolVar(&f.CreateWallet, "cw", false, "")
	flag.BoolVar(&f.ShowWallets, "sw", false, "")
	flag.BoolVar(&f.StartNode, "sn", false, "")
	flag.BoolVar(&f.StartFullNode, "sfn", false, "")
	flag.BoolVar(&f.StartMiningNode, "smn", false, "")

	flag.Parse()
}

func (f *FlagsCLI) PrintUsage() {
	fmt.Println("Usage:")
	fmt.Println("  -s FROM_ADDR TO_ADDR COINS: send coins")
	fmt.Println("  -p: print all the blocks of the blockchain")
	fmt.Println("  -b ADDR: get balance")
	fmt.Println("  -cw: create wallet")
	fmt.Println("  -sw: show wallet")
	fmt.Println("  -sn: start node")
	fmt.Println("  -sfn: start full node")
	fmt.Println("  -smn: start mining node")
}
