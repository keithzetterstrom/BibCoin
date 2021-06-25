package network

import (
	"encoding/hex"
	bcpkg "github.com/keithzetterstrom/BibCoin/internal/pkg/blockchain"
	"log"
)

const typeTx = "tx"
const txInPool = 1

type tx struct {
	AddFrom     string
	Transaction []byte
}

// SendTx sends commandTx request with transaction
func (n *Network) SendTx(addr string, tnx *bcpkg.Transaction) {
	data := tx{AddFrom: n.NetAddr, Transaction: tnx.Serialize()}
	payload := gobEncode(data)
	request := append(commandToBytes(commandTx), payload...)

	n.sendData(addr, request)
}

// handleTx handles request with transaction
// and put it to mem pool with transactions
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
