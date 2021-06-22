package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"log"
	"time"
)

type Block struct {
	Timestamp     int64
	PrevBlockHash []byte
	Hash          []byte
	Nonce         int
	Height        int
	MinerAddress string
}

type ExtensionBlock struct {
	Block
	Transactions    []*Transaction
	StakeholderHash []byte
}

func NewBlock(prevBlockHash []byte, height int, address string) *Block {
	block := &Block{
		Timestamp: time.Now().Unix(),
		MinerAddress: address,
		PrevBlockHash: prevBlockHash,
		Height: height,
	}

	pow := NewProofOfWork(block)
	nonce, hash := pow.Run()

	block.Hash = hash[:]
	block.Nonce = nonce

	return block
}

func NewExtensionBlock(transactions []*Transaction, block *Block) *ExtensionBlock {
	extensionBlock := &ExtensionBlock{
		Block: *block,
		Transactions: transactions,
	}

	extensionBlock.StakeholderHash = []byte("hash[:]")

	return extensionBlock
}

func (b *ExtensionBlock) Serialize() []byte {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)

	err := encoder.Encode(b)
	if err != nil {
		log.Panic(err)
	}

	return result.Bytes()
}

func (b *Block) Serialize() []byte {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)

	err := encoder.Encode(b)
	if err != nil {
		log.Panic(err)
	}

	return result.Bytes()
}

func DeserializeBlock(d []byte) (*Block, error) {
	var block Block

	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&block)
	if err != nil {
		return nil, err
	}

	return &block, nil
}

func DeserializeExtensionBlock(d []byte) (*ExtensionBlock, error) {
	var block ExtensionBlock

	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&block)
	if err != nil {
		return nil, err
	}

	return &block, nil
}

func (b *ExtensionBlock) HashTransactions() []byte {
	var txHashes [][]byte
	var txHash [32]byte

	for _, tx := range b.Transactions {
		txHashes = append(txHashes, tx.ID)
	}
	txHash = sha256.Sum256(bytes.Join(txHashes, []byte{}))

	return txHash[:]
}
