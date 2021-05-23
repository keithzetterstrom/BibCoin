package network

import (
	"fmt"
	bcpkg "github.com/keithzetterstrom/BibCoin/internal/pkg/blockchain"
	"log"
)

const typeBlock = "block"

type block struct {
	AddrFrom string
	Block    []byte
}

type getBlocks struct {
	AddrFrom string
}

func (n *Network) sendBlock(addr string, b *bcpkg.Block) {
	data := block{n.NetAddr, b.Serialize()}
	payload := gobEncode(data)
	request := append(commandToBytes(commandBlock), payload...)

	n.sendData(addr, request)
}

func (n *Network) sendGetBlocks(address string) {
	payload := gobEncode(getBlocks{n.NetAddr})
	request := append(commandToBytes(commandGetBlocks), payload...)

	n.sendData(address, request)
}

func (n *Network) handleBlock(request []byte) {
	var payload block

	err := getDataFromRequest(request, &payload)
	if err != nil {
		log.Panic(err)
	}

	blockData := payload.Block
	block, err := bcpkg.DeserializeBlock(blockData)
	if err != nil {
		log.Println(err)
		return
	}

	fmt.Println("Received a new block!")
	n.Bc.AddBlock(block)

	fmt.Printf("Added block %x\n", block.Hash)

	if len(n.blocksInTransit) > 0 {
		blockHash := n.blocksInTransit[0]
		n.sendGetData(payload.AddrFrom, typeBlock, blockHash)

		n.blocksInTransit = n.blocksInTransit[1:]
	}
}

func (n *Network) handleGetBlocks(request []byte) {
	var payload getBlocks

	err := getDataFromRequest(request, &payload)
	if err != nil {
		log.Panic(err)
	}

	blocks := n.Bc.GetBlockHashes()
	n.sendInv(payload.AddrFrom, typeBlock, blocks)
}
