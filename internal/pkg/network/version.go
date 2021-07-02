package network

import (
	"log"
)

const nodeVersion = 1

type version struct {
	Version    int
	BestHeight int
	AddrFrom   string
}

// sendVersion sends commandVersion request with
// actual height of blockchain in current node
func (n *Network) sendVersion(addr string) {
	bestHeight, err := n.Bc.GetBestHeight()
	if err != nil {
		bestHeight = -1
	}

	bestHeight = bestHeight - len(n.blocksInTransit)

	payload := gobEncode(version{Version: nodeVersion, BestHeight: bestHeight, AddrFrom: n.NetAddr})

	request := append(commandToBytes(commandVersion), payload...)

	n.sendData(addr, request)
}

// sendOK sends commandOK
func (n *Network) sendOK(addr string) {
	n.sendData(addr, commandToBytes(commandOK))
}

// handleVersion handles request with version of other node
// and compares it with current node version
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
