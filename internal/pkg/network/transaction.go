package network

import (
	"encoding/hex"
	"fmt"
	bcpkg "github.com/keithzetterstrom/BibCoin/internal/pkg/blockchain"
	"log"
)

const typeTx = "tx"
const txInPool = 1

type tx struct {
	AddFrom     string
	Transaction []byte
}

func (n *Network) SendTx(addr string, tnx *bcpkg.Transaction) {
	data := tx{AddFrom: n.NetAddr, Transaction: tnx.Serialize()}
	payload := gobEncode(data)
	request := append(commandToBytes(commandTx), payload...)

	fmt.Println("sendTx")
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
}
