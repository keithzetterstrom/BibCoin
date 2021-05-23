package network

import (
	"encoding/hex"
	"fmt"
	bcpkg "github.com/keithzetterstrom/BibCoin/internal/pkg/blockchain"
	"log"
)

const typeTx = "tx"

type tx struct {
	AddFrom     string
	Transaction []byte
}

func (n *Network) SendTx(addr string, tnx *bcpkg.Transaction) {
	data := tx{AddFrom: n.NetAddr, Transaction: tnx.Serialize()}
	payload := gobEncode(data)
	request := append(commandToBytes(commandTx), payload...)

	n.sendData(addr, request)
}

func (n *Network) handleTx(request []byte) {
	var payload tx

	err := getDataFromRequest(request, &payload)
	if err != nil {
		log.Panic(err)
	}

	txData := payload.Transaction
	tx := bcpkg.DeserializeTransaction(txData)
	n.memPool[hex.EncodeToString(tx.ID)] = tx

	if n.NetAddr == fullNodeAddress {
		for _, node := range n.KnownNodes {
			if node != n.NetAddr && node != payload.AddFrom {
				n.sendInv(node, typeTx, [][]byte{tx.ID})
				return
			}
		}
	}

	if len(n.memPool) >= 1 && len(n.Address) > 0 {
	MineTransactions:
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

		cbTx := bcpkg.NewCoinbaseTX(n.Address, "")
		txs = append(txs, cbTx)

		newBlock := n.Bc.MineBlock(txs)

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

		if len(n.memPool) > 0 {
			goto MineTransactions
		}
	}
}
