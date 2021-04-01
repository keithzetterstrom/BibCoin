package iterator

import (
	"github.com/boltdb/bolt"
	"github.com/keithzetterstrom/BibCoin/internal/pkg/blockchain"
	"log"
)

type Iterator struct {
	currentHash []byte
	db          *bolt.DB
}

func NewIterator(bc *blockchain.Blockchain) *Iterator {
	bci := &Iterator{
		currentHash: bc.Tip,
		db: bc.Db,
	}

	return bci
}

func (i *Iterator) Next() *blockchain.Block {
	var block *blockchain.Block

	err := i.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blockchain.BlocksBucket))
		encodedBlock := b.Get(i.currentHash)
		block = blockchain.DeserializeBlock(encodedBlock)

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	i.currentHash = block.PrevBlockHash

	return block
}
