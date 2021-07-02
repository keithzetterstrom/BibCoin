package blockchain

import (
	"github.com/boltdb/bolt"
	"log"
)

type Iterator struct {
	currentHash []byte
	db          *bolt.DB
}

// NewIterator returns Iterator to iterate over the Blockchain
func (bc *Blockchain) NewIterator() *Iterator {
	bci := &Iterator{
		currentHash: bc.Tip,
		db: bc.Db,
	}

	return bci
}

// Next returns next ExtensionBlock in Blockchain
func (i *Iterator) Next() *ExtensionBlock {
	var block *ExtensionBlock
	var err error

	err = i.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BlocksBucket))
		encodedBlock := b.Get(i.currentHash)
		block, err = DeserializeExtensionBlock(encodedBlock)
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
