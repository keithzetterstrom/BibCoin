package cli

import (
	"flag"
	"fmt"
)

type FlagsCLI struct {
	SendCmd   string
	PrintChainCmd bool
	BalanceCmd string
}

func NewFlagCLI() *FlagsCLI {
	return &FlagsCLI{}
}

func (f *FlagsCLI) FlagsCLI()  {
	flag.StringVar(&f.SendCmd, "s", "", "")
	flag.BoolVar(&f.PrintChainCmd, "p", false, "")
	flag.StringVar(&f.BalanceCmd, "b", "", "")

	flag.Parse()
}

func (f *FlagsCLI) PrintUsage() {
	fmt.Println("Usage:")
	fmt.Println("  -s FROM_ADDR TO_ADDR COINS: send coins")
	fmt.Println("  -p:	print all the blocks of the blockchain")
	fmt.Println("  -b ADDR: get balance")
}
