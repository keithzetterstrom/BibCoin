package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"io"
	"log"
	"os"
)

const errorDataBaseNotExist = "database is not exists"

type Blockchain struct {
	Tip []byte
	Db  *bolt.DB
}

func newGenesisBlock(coinbase *Transaction) *Block {
	return NewBlock([]*Transaction{coinbase}, []byte{}, 1)
}

func (bc *Blockchain) GetBlock(blockHash []byte) (Block, error) {
	var block *Block
	var err error

	err = bc.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BlocksBucket))

		blockData := b.Get(blockHash)

		if blockData == nil {
			return errors.New("Block is not found. ")
		}

		block, err = DeserializeBlock(blockData)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return *block, err
	}

	return *block, nil
}

func (bc *Blockchain) GetBlockHashes() [][]byte {
	var blocks [][]byte
	bci := bc.NewIterator()

	for {
		block := bci.Next()

		blocks = append(blocks, block.Hash)

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return blocks
}

func (bc *Blockchain) MineBlock(transactions []*Transaction) *Block {
	var lastHash []byte
	var lastHeight int

	var validTx []*Transaction

	for _, tx := range transactions {
		if !bc.VerifyTransaction(tx) {
			log.Println("Invalid transaction")
			continue
		}
		validTx = append(validTx, tx)
	}

	err := bc.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BlocksBucket))
		lastHash = b.Get([]byte("l"))

		blockData := b.Get(lastHash)
		block, err := DeserializeBlock(blockData)
		if err != nil {
			return err
		}

		lastHeight = block.Height

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	newBlock := NewBlock(validTx, lastHash, lastHeight + 1)

	err = bc.Db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BlocksBucket))
		err := b.Put(newBlock.Hash, newBlock.Serialize())
		if err != nil {
			log.Panic(err)
		}

		err = b.Put([]byte("l"), newBlock.Hash)
		if err != nil {
			log.Panic(err)
		}

		bc.Tip = newBlock.Hash

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	return newBlock
}

func (bc *Blockchain) AddBlock(block *Block) error {
	pow := NewProofOfWork(block)
	pow.prepareData(block.Nonce)
	if !pow.Validate() {
		return errors.New("Block invalid ")
	}

	err := bc.Db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BlocksBucket))
		blockInDb := b.Get(block.Hash)

		if blockInDb != nil {
			return errors.New("Block already exists ")
		}

		blockData := block.Serialize()
		err := b.Put(block.Hash, blockData)
		if err != nil {
			log.Panic(err)
		}

		lastHash := b.Get([]byte("l"))
		lastBlockData := b.Get(lastHash)
		lastBlock, err := DeserializeBlock(lastBlockData)
		if errors.Is(err, io.EOF) {
			err = b.Put([]byte("l"), block.Hash)
			if err != nil {
				log.Panic(err)
			}
			bc.Tip = block.Hash
			return nil
		}
		if err != nil {
			log.Panic(err)
		}

		if block.Height > lastBlock.Height {
			err = b.Put([]byte("l"), block.Hash)
			if err != nil {
				log.Panic(err)
			}
			bc.Tip = block.Hash
			return nil
		}

		return nil
	})
	return err
}

func (bc *Blockchain) AddGenesisBlock(address string)  {
	err := bc.Db.Update(func(tx *bolt.Tx) error {
		cbtx := NewCoinbaseTX(address, genesisCoinbaseData)
		genesis := newGenesisBlock(cbtx)

		b := tx.Bucket([]byte(BlocksBucket))

		err := b.Put(genesis.Hash, genesis.Serialize())
		if err != nil {
			log.Panic(err)
		}

		err = b.Put([]byte("l"), genesis.Hash)
		if err != nil {
			log.Panic(err)
		}
		bc.Tip = genesis.Hash

		return nil
	})

	if err != nil {
		log.Panic(err)
	}
}


func (bc *Blockchain) FindTransaction(ID []byte) (Transaction, error) {
	bci := bc.NewIterator()

	for {
		block := bci.Next()

		for _, tx := range block.Transactions {
			if bytes.Compare(tx.ID, ID) == 0 {
				return *tx, nil
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return Transaction{}, errors.New("Transaction is not found ")
}

func (bc *Blockchain) FindUnspentTransactions(pubKeyHash []byte) []Transaction {
	var unspentTransactions []Transaction
	spentTXOs := make(map[string][]int)
	bci := bc.NewIterator()

	for {
		block := bci.Next()

		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)

		Outputs:
			for outIdx, out := range tx.Vout {
				if spentTXOs[txID] != nil {
					for _, spentOut := range spentTXOs[txID] {
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}

				if out.IsLockedWithKey(pubKeyHash) {
					unspentTransactions = append(unspentTransactions, *tx)
				}
			}

			if tx.IsCoinbase() == false {
				for _, in := range tx.Vin {
					if in.UsesKey(pubKeyHash) {
						inTxID := hex.EncodeToString(in.OutTxID)
						spentTXOs[inTxID] = append(spentTXOs[inTxID], in.OutIndex)
					}
				}
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return unspentTransactions
}

func (bc *Blockchain) FindUnspentTxOutputs(pubKeyHash []byte) []TXOutput {
	var txOutputs []TXOutput
	unspentTransactions := bc.FindUnspentTransactions(pubKeyHash)

	for _, tx := range unspentTransactions {
		for _, out := range tx.Vout {
			if out.IsLockedWithKey(pubKeyHash) {
				txOutputs = append(txOutputs, out)
			}
		}
	}

	return txOutputs
}

func (bc *Blockchain) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int) {
	unspentOutputs := make(map[string][]int)
	unspentTXs := bc.FindUnspentTransactions(pubKeyHash)
	accumulated := 0

Work:
	for _, tx := range unspentTXs {
		txID := hex.EncodeToString(tx.ID)

		for outIdx, out := range tx.Vout {
			if out.IsLockedWithKey(pubKeyHash) && accumulated < amount {
				accumulated += out.Value
				unspentOutputs[txID] = append(unspentOutputs[txID], outIdx)

				if accumulated >= amount {
					break Work
				}
			}
		}
	}

	return accumulated, unspentOutputs
}

func dbExists(nodeID string) bool {
	dbFile := fmt.Sprintf(dbFile, nodeID)
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}

	return true
}

func (bc *Blockchain) SignTransaction(tx *Transaction, privKey ecdsa.PrivateKey) {
	prevTXs := make(map[string]Transaction)

	for _, vin := range tx.Vin {
		prevTX, err := bc.FindTransaction(vin.OutTxID)
		if err != nil {
			log.Panic(err)
		}
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	tx.Sign(privKey, prevTXs)
}

func (bc *Blockchain) VerifyTransaction(tx *Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}

	prevTXs := make(map[string]Transaction)

	for _, vin := range tx.Vin {
		prevTX, err := bc.FindTransaction(vin.OutTxID)
		if err != nil {
			log.Panic(err)
		}
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	return tx.Verify(prevTXs)
}

func (bc *Blockchain) GetBestHeight() (int, error) {
	var lastBlock *Block
	var err error

	err = bc.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BlocksBucket))
		lastHash := b.Get([]byte("l"))
		blockData := b.Get(lastHash)
		lastBlock, err = DeserializeBlock(blockData)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	return lastBlock.Height, nil
}

func NewBlockchain(nodeID string) (*Blockchain, error) {
	dbFile := fmt.Sprintf(dbFile, nodeID)
	if !dbExists(nodeID) {
		return nil, errors.New(errorDataBaseNotExist)
	}

	var tip []byte
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BlocksBucket))
		tip = b.Get([]byte("l"))

		return nil
	})
	if err != nil {
		log.Println("Blockchain is empty")
	}

	bc := Blockchain{Tip: tip, Db: db}

	return &bc, nil
}

func CreateBlockchain(address string, nodeID string) *Blockchain {
	if dbExists(nodeID) {
		log.Println("Blockchain already exists.")
		os.Exit(1)
	}
	dbFile := fmt.Sprintf(dbFile, nodeID)

	var tip []byte
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		cbtx := NewCoinbaseTX(address, genesisCoinbaseData)
		genesis := newGenesisBlock(cbtx)

		b, err := tx.CreateBucket([]byte(BlocksBucket))
		if err != nil {
			log.Panic(err)
		}

		err = b.Put(genesis.Hash, genesis.Serialize())
		if err != nil {
			log.Panic(err)
		}

		err = b.Put([]byte("l"), genesis.Hash)
		if err != nil {
			log.Panic(err)
		}
		tip = genesis.Hash

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	bc := Blockchain{tip, db}

	return &bc
}

func CreateEmptyBlockchain(nodeID string) *Blockchain {
	if dbExists(nodeID) {
		log.Println("Blockchain already exists.")
		os.Exit(1)
	}
	dbFile := fmt.Sprintf(dbFile, nodeID)

	var tip []byte
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket([]byte(BlocksBucket))
		if err != nil {
			log.Panic(err)
		}

		return nil
	})

	bc := Blockchain{tip, db}

	return &bc
}