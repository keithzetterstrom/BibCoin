package network

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	bcpkg "github.com/keithzetterstrom/BibCoin/internal/pkg/blockchain"
	"io"
	"io/ioutil"
	"log"
	"net"
)

const (
	commandOK        = "ok"
	commandVersion   = "version"
	commandTx        = "tx"
	commandBlock     = "block"
	commandInv       = "inv"
	commandGetData   = "getdata"
	commandGetBlocks = "getblocks"
)

const protocol = "tcp"
const commandLength = 12
const fullNodeAddress = "127.0.0.1:9000"

type getData struct {
	AddrFrom string
	Type     string
	ID       []byte
}

type Network struct {
	NetAddr         string
	Address         string
	Bc              *bcpkg.Blockchain
	KnownNodes      []string
	memPool         map[string]bcpkg.Transaction
	blocksInTransit [][]byte
}

func NewNetwork(bc *bcpkg.Blockchain, netAddress, address string) *Network {
	return &Network{
		Bc: bc,
		NetAddr: netAddress,
		Address: address,
		KnownNodes: []string{fullNodeAddress},
		memPool: make(map[string]bcpkg.Transaction),
		blocksInTransit: [][]byte{},
	}
}

func commandToBytes(command string) []byte {
	var commandBytes[commandLength]byte

	for i, c := range command {
		commandBytes[i] = byte(c)
	}

	return commandBytes[:]
}

func bytesToCommand(bytes []byte) string {
	var command []byte

	for _, b := range bytes {
		if b != 0x0 {
			command = append(command, b)
		}
	}

	return fmt.Sprintf("%s", command)
}

func (n *Network) requestBlocks() {
	for _, node := range n.KnownNodes {
		n.sendGetBlocks(node)
	}
}

func (n *Network) sendData(addr string, data []byte) {
	conn, err := net.Dial(protocol, addr)
	if err != nil {
		fmt.Printf("%s is not available\n", addr)
		var updatedNodes []string

		for _, node := range n.KnownNodes {
			if node != addr {
				updatedNodes = append(updatedNodes, node)
			}
		}

		n.KnownNodes = updatedNodes

		return
	}
	defer conn.Close()

	_, err = io.Copy(conn, bytes.NewReader(data))
	if err != nil {
		log.Panic(err)
	}
}

func (n *Network) sendGetData(address, kind string, id []byte) {
	payload := gobEncode(getData{AddrFrom: n.NetAddr, Type: kind, ID: id})
	request := append(commandToBytes(commandGetData), payload...)

	fmt.Println("sendGetData")
	n.sendData(address, request)
}

func (n *Network) handleGetData(request []byte) {
	var payload getData

	err := getDataFromRequest(request, &payload)
	if err != nil {
		log.Panic(err)
	}

	if payload.Type == typeBlock {
		block, err := n.Bc.GetBlock(payload.ID)
		if err != nil {
			return
		}

		n.sendBlock(payload.AddrFrom, &block)
	}

	if payload.Type == typeTx {
		txID := hex.EncodeToString(payload.ID)
		tx := n.memPool[txID]

		n.SendTx(payload.AddrFrom, &tx)
	}
}

func (n *Network) handleConnection(conn net.Conn) bool {
	defer conn.Close()

	request, err := ioutil.ReadAll(conn)
	if err != nil {
		log.Println(err)
		return false
	}

	command := bytesToCommand(request[:commandLength])
	fmt.Printf("Received %s command\n", command)

	switch command {
	case commandBlock:
		n.handleBlock(request)
	case commandInv:
		n.handleInv(request)
	case commandGetBlocks:
		n.handleGetBlocks(request)
	case commandGetData:
		n.handleGetData(request)
	case commandTx:
		n.handleTx(request)
	case commandVersion:
		n.handleVersion(request)
	case commandOK:
		fmt.Println("Every thing update")
		return true
	default:
		fmt.Println("Unknown command!")
	}

	return false
}

func (n *Network) StartServer() {
	ln, err := net.Listen(protocol, n.NetAddr)
	if err != nil {
		log.Println(err)
		return
	}
	defer ln.Close()

	n.sendVersion(fullNodeAddress)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			return
		}
		if n.handleConnection(conn) {
			return
		}
		n.sendVersion(fullNodeAddress)
	}
}

func (n *Network) StartMineServer() {
	ln, err := net.Listen(protocol, n.NetAddr)
	if err != nil {
		log.Println(err)
		return
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			return
		}
		n.handleConnection(conn)
	}
}

func (n *Network) StartFullServer() {
	ln, err := net.Listen(protocol, n.NetAddr)
	if err != nil {
		log.Println(err)
		return
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			return
		}
		n.handleConnection(conn)
	}
}

func gobEncode(data interface{}) []byte {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(data)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}

func (n *Network) nodeIsKnown(addr string) bool {
	for _, node := range n.KnownNodes {
		if node == addr {
			return true
		}
	}

	return false
}

func getDataFromRequest(request []byte, payload interface{}) (err error) {
	var buff bytes.Buffer
	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err = dec.Decode(payload)
	if err != nil {
		return err
	}

	return  nil
}
