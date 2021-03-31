package cli

import (
	"flag"
	"fmt"
)

type FlagsCLI struct {
	AddBlockCmd   string
	PrintChainCmd bool
}

func NewFlagCLI() *FlagsCLI {
	return &FlagsCLI{}
}

func (f *FlagsCLI) FlagsCLI()  {
	flag.StringVar(&f.AddBlockCmd, "a", "", "")
	flag.BoolVar(&f.PrintChainCmd, "p", false, "")

	flag.Parse()
}

func (f *FlagsCLI) PrintUsage() {
	fmt.Println("Usage:")
	fmt.Println("  -a BLOCK_DATA: add a block to the blockchain")
	fmt.Println("  -p:	print all the blocks of the blockchain")
}
