package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
)

type Transaction struct {
	ID   []byte
	Vin  []TXInput
	Vout []TXOutput
}

type TXOutput struct {
	Value        int
	ScriptPubKey string
}

type TXInput struct {
	OutTxID   []byte
	OutIndex  int
	ScriptSig string
}

func (tx Transaction) IsCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].OutTxID) == 0 && tx.Vin[0].OutIndex == -1
}

func (in *TXInput) CanUnlockOutputWith(unlockingData string) bool {
	return in.ScriptSig == unlockingData
}

func (out *TXOutput) CanBeUnlockedWith(unlockingData string) bool {
	return out.ScriptPubKey == unlockingData
}

func NewCoinbaseTX(to, data string) *Transaction {
	if data == "" {
		data = fmt.Sprintf("Reward to '%s'", to)
	}

	txin := TXInput{
		OutTxID:   []byte{},
		OutIndex:  -1,
		ScriptSig: data,
	}
	txout := TXOutput{
		Value:        subsidy,
		ScriptPubKey: to,
	}
	tx := Transaction{
		Vin:  []TXInput{txin},
		Vout: []TXOutput{txout},
	}
	tx.SetID()

	return &tx
}

func NewTransaction(from, to string, amount int, bc *Blockchain) *Transaction {
	var inputs []TXInput
	var outputs []TXOutput

	acc, validOutputs := bc.FindSpendableOutputs(from, amount)

	if acc < amount {
		log.Panic("ERROR: Not enough funds")
	}

	for rawTxID, outs := range validOutputs {
		txID, err := hex.DecodeString(rawTxID)
		if err != nil {
			log.Panic(err)
		}

		for _, out := range outs {
			input := TXInput{
				OutTxID: txID,
				OutIndex: out,
				ScriptSig: from,
			}
			inputs = append(inputs, input)
		}
	}

	outputs = append(outputs, TXOutput{Value: amount, ScriptPubKey: to})
	if acc > amount {
		outputs = append(outputs, TXOutput{Value: acc - amount, ScriptPubKey: from})
	}

	tx := Transaction{Vin: inputs, Vout: outputs}
	tx.SetID()

	return &tx
}

func (tx *Transaction) SetID() {
	var encoded bytes.Buffer
	var hash [32]byte

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	if err != nil {
		log.Panic(err)
	}
	hash = sha256.Sum256(encoded.Bytes())
	tx.ID = hash[:]
}