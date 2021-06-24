package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/keithzetterstrom/BibCoin/tools/base58"
	"io"
	"log"
	"os"
)

const (
	errorDataBaseNotExist         = "database is not exists"
	errorStakeholderIndexNotFound = "Stakeholder index not found "
)

type Blockchain struct {
	Tip []byte
	Db  *bolt.DB
}

func newGenesisBlock(coinbase *Transaction) *ExtensionBlock {
	block := NewBlock([]byte{}, 1, "")
	return NewExtensionBlock([]*Transaction{coinbase}, block)
}

func (bc *Blockchain) GetBlock(blockHash []byte) (ExtensionBlock, error) {
	var block *ExtensionBlock
	var err error

	err = bc.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BlocksBucket))

		blockData := b.Get(blockHash)

		if blockData == nil {
			return errors.New("Block is not found. ")
		}

		block, err = DeserializeExtensionBlock(blockData)
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

func (bc *Blockchain) MineBlock(minerAddress string) *Block {
	var lastHash []byte
	var lastHeight int

	// находим последний хнш и высоту относительно генезис блока
	err := bc.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BlocksBucket))
		lastHash = b.Get([]byte("l"))

		blockData := b.Get(lastHash)
		block, err := DeserializeExtensionBlock(blockData)
		if err != nil {
			return err
		}

		lastHeight = block.Height

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	newBlock := NewBlock(lastHash, lastHeight + 1, minerAddress)

	return newBlock
}

func (bc *Blockchain) AddBlock(block *ExtensionBlock) error {
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
		lastBlock, err := DeserializeExtensionBlock(lastBlockData)
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

func (bc *Blockchain) AddNewBlock(newBlock *Block, transactions []*Transaction, address string) (*ExtensionBlock, error) {
	// проверяем работу майнера
	pow := NewProofOfWork(newBlock)
	pow.prepareData(newBlock.Nonce)
	if !pow.Validate() {
		return nil, errors.New("Block invalid ")
	}

	// проверяем, является ли стейклолдер избранным
	lastIndex, err := bc.GetLastSatoshiIndex()
	if err != nil {
		return nil, fmt.Errorf("Failed to add new block: %s ", err)
	}
	stakeholderIndex := GetStakeholderIndexByHash(newBlock.Hash, lastIndex)

	pubKeyHash := base58.DecodeBase58([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash) - 4]

	if !bc.checkStakeholderIndex(stakeholderIndex, pubKeyHash) {
		return nil, errors.New(errorStakeholderIndexNotFound)
	}

	// проверяем транзакции перед записью в блок
	var validTx []*Transaction

	subsidyArr := make([]int, subsidy)
	for i := 0; i < subsidy; i++  {
		subsidyArr[i] = i + lastIndex
	}

	for _, tx := range transactions {
		if !bc.VerifyTransaction(tx) {
			log.Println("Invalid transaction")
			continue
		}
		validTx = append(validTx, tx)
	}

	extensionBlock := NewExtensionBlock(validTx, newBlock)

	// добавляем новый блок в бд
	err = bc.Db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BlocksBucket))
		err := b.Put(extensionBlock.Hash, extensionBlock.Serialize())
		if err != nil {
			log.Panic(err)
		}

		err = b.Put([]byte("l"), extensionBlock.Hash)
		if err != nil {
			log.Panic(err)
		}

		bc.Tip = extensionBlock.Hash

		return nil
	})
	if err != nil {
		return nil, err
	}

	return extensionBlock, nil
}

func (bc *Blockchain) AddGenesisBlock(address string) {
	err := bc.Db.Update(func(tx *bolt.Tx) error {
		lastIndex, err := bc.GetLastSatoshiIndex()
		if err != nil {
			fmt.Println(err)
		}
		cbtx := NewCoinbaseTX(address, address, genesisCoinbaseData, lastIndex)
		genesis := newGenesisBlock(cbtx)

		b := tx.Bucket([]byte(BlocksBucket))

		err = b.Put(genesis.Hash, genesis.Serialize())
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

func (bc *Blockchain) FindSpendableOutputs(pubKeyHash []byte, amount int) ([]int, map[string][]int) {
	unspentOutputs := make(map[string][]int)
	unspentTXs := bc.FindUnspentTransactions(pubKeyHash)
	var accumulated []int

Work:
	for _, tx := range unspentTXs {
		txID := hex.EncodeToString(tx.ID)

		for outIdx, out := range tx.Vout {
			if out.IsLockedWithKey(pubKeyHash) && len(accumulated) < amount {
				accumulated = append(accumulated, out.Value...)
				unspentOutputs[txID] = append(unspentOutputs[txID], outIdx)

				if len(accumulated) >= amount {
					break Work
				}
			}
		}
	}

	return accumulated, unspentOutputs
}

func dbExists(dbFile string) bool {
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
	var lastBlock *ExtensionBlock
	var err error

	err = bc.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BlocksBucket))
		lastHash := b.Get([]byte("l"))
		blockData := b.Get(lastHash)
		lastBlock, err = DeserializeExtensionBlock(blockData)
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

func (bc *Blockchain) GetLastSatoshiIndex() (int, error) {
	var lastBlock *ExtensionBlock
	var err error

	err = bc.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BlocksBucket))
		lastHash := b.Get([]byte("l"))
		blockData := b.Get(lastHash)
		lastBlock, err = DeserializeExtensionBlock(blockData)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	maxIndex := 0
	for _, tx := range lastBlock.Transactions {
		if tx.IsCoinbase() {
			for _, txOut := range tx.Vout{
				max := txOut.Value.GetMaxIndex(txOut.Value)
				if maxIndex < max {
					maxIndex = max
				}
			}
		}
	}

	return maxIndex + 1, nil
}

func (bc *Blockchain) checkStakeholderIndex(stakeholderIndex int, pubKeyHash []byte) bool {
	unspentTxOutputs := bc.FindUnspentTxOutputs(pubKeyHash)

	for _, out := range unspentTxOutputs {
		if out.Value.FindIndex(out.Value, stakeholderIndex) {
			return true
		}
	}
	return false
}

func NewBlockchain(dbFile string) (*Blockchain, error) {
	if !dbExists(dbFile) {
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

func CreateBlockchain(address string, dbFile string) *Blockchain {
	if dbExists(dbFile) {
		log.Println("Blockchain already exists.")
		os.Exit(1)
	}

	var tip []byte
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		cbtx := NewCoinbaseTX(address, address, genesisCoinbaseData, 0)
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

func CreateEmptyBlockchain(dbFile string) *Blockchain {
	if dbExists(dbFile) {
		log.Println("Blockchain already exists.")
		os.Exit(1)
	}

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
