package network

import (
	"log"
)

type version struct {
	Version    int
	BestHeight int
	AddrFrom   string
}

func (n *Network) sendVersion(addr string) {
	bestHeight, err := n.Bc.GetBestHeight()
	if err != nil {
		bestHeight = -1
	}

	payload := gobEncode(version{Version: nodeVersion, BestHeight: bestHeight, AddrFrom: n.NetAddr})

	request := append(commandToBytes(commandVersion), payload...)

	n.sendData(addr, request)
}

func (n *Network) sendOK(addr string) {
	n.sendData(addr, commandToBytes(commandOK))
}

func (n *Network) handleVersion(request []byte) {
	var payload version

	err := getDataFromRequest(request, &payload)
	if err != nil {
		log.Panic(err)
	}

	myBestHeight, _ := n.Bc.GetBestHeight()

	foreignerBestHeight := payload.BestHeight

	if myBestHeight < foreignerBestHeight {
		n.sendGetBlocks(payload.AddrFrom)
	} else if myBestHeight > foreignerBestHeight {
		n.sendVersion(payload.AddrFrom)
	} else {
		n.sendOK(payload.AddrFrom)
	}

	if !n.nodeIsKnown(payload.AddrFrom) {
		n.KnownNodes = append(n.KnownNodes, payload.AddrFrom)
	}
}
