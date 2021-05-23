package blockchain

import (
	"github.com/boltdb/bolt"
	"log"
)

type Iterator struct {
	currentHash []byte
	db          *bolt.DB
}

func (bc *Blockchain) NewIterator() *Iterator {
	bci := &Iterator{
		currentHash: bc.Tip,
		db: bc.Db,
	}

	return bci
}

func (i *Iterator) Next() *Block {
	var block *Block
	var err error

	err = i.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BlocksBucket))
		encodedBlock := b.Get(i.currentHash)
		block, err = DeserializeBlock(encodedBlock)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	i.currentHash = block.PrevBlockHash

	return block
}
