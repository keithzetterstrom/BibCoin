package network

import (
	"bytes"
	"encoding/hex"
	"log"
)

type inv struct {
	AddrFrom string
	Type     string
	Items    [][]byte
}

// sendInv sends commandInv request with kind of needed data
// and ids of the data
func (n *Network) sendInv(address, kind string, items [][]byte) {
	inventory := inv{AddrFrom: n.NetAddr, Type: kind, Items: items}
	payload := gobEncode(inventory)
	request := append(commandToBytes(commandInv), payload...)

	n.sendData(address, request)
}

// handleInv handles inventory request, detects its type
// and sends get needed data request
func (n *Network) handleInv(request []byte) {
	var payload inv

	err := getDataFromRequest(request, &payload)
	if err != nil {
		log.Panic(err)
	}

	if payload.Type == typeBlock {
		n.blocksInTransit = payload.Items

		blockHash := payload.Items[0]

		n.sendGetData(payload.AddrFrom, typeBlock, blockHash)

		newInTransit := [][]byte{}
		for _, b := range n.blocksInTransit {
			if bytes.Compare(b, blockHash) != 0 {
				newInTransit = append(newInTransit, b)
			}
		}
		n.blocksInTransit = newInTransit
	}

	if payload.Type == typeTx {
		txID := payload.Items[0]

		if n.memPool[hex.EncodeToString(txID)].ID == nil {
			n.sendGetData(payload.AddrFrom, typeTx, txID)
		}
	}
}
