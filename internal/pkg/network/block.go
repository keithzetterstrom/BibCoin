package network

import (
	"encoding/hex"
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

// sendBlock sends commandBlock request with block
func (n *Network) sendBlock(addr string, b *bcpkg.ExtensionBlock) {
	data := block{n.NetAddr, b.Serialize()}
	payload := gobEncode(data)
	request := append(commandToBytes(commandBlock), payload...)

	n.sendData(addr, request)
}

// sendBlock sends commandGetBlocks request
func (n *Network) sendGetBlocks(address string) {
	payload := gobEncode(getBlocks{n.NetAddr})
	request := append(commandToBytes(commandGetBlocks), payload...)

	n.sendData(address, request)
}

// sendBlock sends commandNewBlock request with block
// when mined a new block
func (n *Network) sendNewBlock(addr string, b *bcpkg.Block) {
	data := block{n.NetAddr, b.Serialize()}
	payload := gobEncode(data)
	request := append(commandToBytes(commandNewBlock), payload...)

	n.sendData(addr, request)
}

// handleBlock handles block
func (n *Network) handleBlock(request []byte) {
	var payload block

	err := getDataFromRequest(request, &payload)
	if err != nil {
		log.Panic(err)
	}

	blockData := payload.Block
	block, err := bcpkg.DeserializeExtensionBlock(blockData)
	if err != nil {
		log.Println(err)
		return
	}

	err = n.Bc.AddBlock(block)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("Added block %x with high %d \n", block.Hash, block.Height)
	}

	if len(n.memPool) > 0 {
		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)
			if _, ok := n.memPool[txID]; ok {
				delete(n.memPool, txID)
			}
		}
	}

	if len(n.blocksInTransit) > 0 {
		blockHash := n.blocksInTransit[0]
		n.sendGetData(payload.AddrFrom, typeBlock, blockHash)

		n.blocksInTransit = n.blocksInTransit[1:]
	}
}

// handleGetBlocks handles hashes of blocks and sends inventory
func (n *Network) handleGetBlocks(request []byte) {
	var payload getBlocks

	err := getDataFromRequest(request, &payload)
	if err != nil {
		log.Panic(err)
	}

	blocks := n.Bc.GetBlockHashes()
	n.sendInv(payload.AddrFrom, typeBlock, blocks)
}

// handleNewBlock handles new block which be mined by miner
func (n *Network) handleNewBlock(request []byte)  {
	var payload block

	err := getDataFromRequest(request, &payload)
	if err != nil {
		log.Panic(err)
	}

	if len(n.memPool) < txInPool {
		n.sendOK(payload.AddrFrom)
		return
	}

	blockData := payload.Block
	block, err := bcpkg.DeserializeBlock(blockData)
	if err != nil {
		log.Println(err)
		return
	}

	var txs []*bcpkg.Transaction

	for id := range n.memPool {
		tx := n.memPool[id]
		if n.Bc.VerifyTransaction(&tx) {
			txs = append(txs, &tx)
		}
	}

	if len(txs) == 0 {
		fmt.Println("All transactions are invalid")
		return
	}

	lastIndex, err := n.Bc.GetLastSatoshiIndex()
	if err != nil {
		log.Panic(err)
	}

	cbTx := bcpkg.NewCoinbaseTX(block.MinerAddress, n.Address, "", lastIndex)
	txs = append(txs, cbTx)

	newBlock, err := n.Bc.AddNewBlock(block, txs, n.Address)
	if err != nil {
		fmt.Println(err)
		n.sendOK(payload.AddrFrom)
		return
	} else {
		fmt.Printf("Added block %x with high %d \n", block.Hash, block.Height)
	}

	fmt.Println("New block is mined!")

	for _, tx := range txs {
		txID := hex.EncodeToString(tx.ID)
		delete(n.memPool, txID)
	}

	for _, node := range n.KnownNodes {
		if node != n.NetAddr {
			n.sendInv(node, typeBlock, [][]byte{newBlock.Hash})
		}
	}
}
