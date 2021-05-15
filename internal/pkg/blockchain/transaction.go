package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	walletpkg "github.com/keithzetterstrom/BibCoin/internal/pkg/wallet"
	"github.com/keithzetterstrom/BibCoin/tools/base58"
	"log"
	"math/big"
)

type Transaction struct {
	ID   []byte
	Vin  []TXInput
	Vout []TXOutput
}

type TXOutput struct {
	Value      int
	PubKeyHash []byte
}

func NewTXOutput(value int, address string) *TXOutput {
	txo := &TXOutput{Value: value}
	txo.Lock([]byte(address))

	return txo
}

type TXInput struct {
	OutTxID   []byte
	OutIndex  int
	Signature []byte
	PubKey    []byte
}

func (tx Transaction) IsCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].OutTxID) == 0 && tx.Vin[0].OutIndex == -1
}

func (in *TXInput) UsesKey(pubKeyHash []byte) bool {
	lockingHash := base58.HashPubKey(in.PubKey)

	return bytes.Compare(lockingHash, pubKeyHash) == 0
}

func (out *TXOutput) Lock(address []byte) {
	pubKeyHash := base58.DecodeBase58(address)
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash) - 4]
	out.PubKeyHash = pubKeyHash
}

func (out *TXOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Compare(out.PubKeyHash, pubKeyHash) == 0
}

func NewCoinbaseTX(to, data string) *Transaction {
	if data == "" {
		data = fmt.Sprintf("Reward to '%s'", to)
	}

	txin := TXInput{
		OutTxID:   []byte{},
		OutIndex:  -1,
		PubKey: []byte(data),
	}
	txout := NewTXOutput(subsidy, to)
	tx := Transaction{
		Vin:  []TXInput{txin},
		Vout: []TXOutput{*txout},
	}
	tx.ID = tx.Hash()

	return &tx
}

func NewTransaction(from, to string, amount int, bc *Blockchain) *Transaction {
	var inputs []TXInput
	var outputs []TXOutput

	wallets, err := walletpkg.NewWallets()
	if err != nil {
		log.Panic(err)
	}
	wallet := wallets.GetWallet(from)
	pubKeyHash := base58.HashPubKey(wallet.PublicKey)

	acc, validOutputs := bc.FindSpendableOutputs(pubKeyHash, amount)

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
				Signature: nil,
				PubKey: wallet.PublicKey,
			}
			inputs = append(inputs, input)
		}
	}

	outputs = append(outputs, *NewTXOutput(amount, to))
	if acc > amount {
		outputs = append(outputs, *NewTXOutput(acc - amount, from))
	}

	tx := Transaction{Vin: inputs, Vout: outputs}
	tx.ID = tx.Hash()

	bc.SignTransaction(&tx, wallet.PrivateKey)

	return &tx
}

func (tx *Transaction) Sign(privatKey ecdsa.PrivateKey, prevTXs map[string]Transaction) {
	if tx.IsCoinbase() {
		return
	}

	for _, vin := range tx.Vin {
		if prevTXs[hex.EncodeToString(vin.OutTxID)].ID == nil {
			log.Panic("ERROR: Previous transaction is not correct")
		}
	}

	txCopy := tx.TrimmedCopy()

	for inID, vin := range txCopy.Vin {
		prevTx := prevTXs[hex.EncodeToString(vin.OutTxID)]
		txCopy.Vin[inID].Signature = nil
		txCopy.Vin[inID].PubKey = prevTx.Vout[vin.OutIndex].PubKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Vin[inID].PubKey = nil

		r, s, err := ecdsa.Sign(rand.Reader, &privatKey, txCopy.ID)
		if err != nil {
			log.Panic(err)
		}
		signature := append(r.Bytes(), s.Bytes()...)

		tx.Vin[inID].Signature = signature
	}
}

func (tx *Transaction) Verify(prevTXs map[string]Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}

	for _, vin := range tx.Vin {
		if prevTXs[hex.EncodeToString(vin.OutTxID)].ID == nil {
			log.Panic("ERROR: Previous transaction is not correct")
		}
	}

	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256()

	for inID, vin := range tx.Vin {
		prevTx := prevTXs[hex.EncodeToString(vin.OutTxID)]
		txCopy.Vin[inID].Signature = nil
		txCopy.Vin[inID].PubKey = prevTx.Vout[vin.OutIndex].PubKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Vin[inID].PubKey = nil

		r := big.Int{}
		s := big.Int{}
		sigLen := len(vin.Signature)
		r.SetBytes(vin.Signature[:(sigLen / 2)])
		s.SetBytes(vin.Signature[(sigLen / 2):])

		x := big.Int{}
		y := big.Int{}
		keyLen := len(vin.PubKey)
		x.SetBytes(vin.PubKey[:(keyLen / 2)])
		y.SetBytes(vin.PubKey[(keyLen / 2):])

		rawPubKey := ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}
		if ecdsa.Verify(&rawPubKey, txCopy.ID, &r, &s) == false {
			return false
		}
	}

	return true
}

func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs []TXInput
	var outputs []TXOutput

	for _, vin := range tx.Vin {
		inputs = append(inputs, TXInput{OutTxID: vin.OutTxID, OutIndex: vin.OutIndex})
	}

	for _, vout := range tx.Vout {
		outputs = append(outputs, TXOutput{Value: vout.Value, PubKeyHash: vout.PubKeyHash})
	}

	txCopy := Transaction{tx.ID, inputs, outputs}

	return txCopy
}

func (tx Transaction) Serialize() []byte {
	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	if err != nil {
		log.Panic(err)
	}

	return encoded.Bytes()
}

func (tx *Transaction) Hash() []byte {
	var hash [32]byte

	txCopy := *tx
	txCopy.ID = []byte{}

	hash = sha256.Sum256(txCopy.Serialize())

	return hash[:]
}
