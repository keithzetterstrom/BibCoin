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
	Value      satoshies
	PubKeyHash []byte
}

// NewTXOutput returns new TXOutput
func NewTXOutput(value satoshies, address string) *TXOutput {
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

// IsCoinbase returns if Transaction is coinbase
func (tx Transaction) IsCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].OutTxID) == 0 && tx.Vin[0].OutIndex == -1
}

// UsesKey returns true if TXInput contains input public key
func (in *TXInput) UsesKey(pubKeyHash []byte) bool {
	lockingHash := base58.HashPubKey(in.PubKey)

	return bytes.Compare(lockingHash, pubKeyHash) == 0
}

// Lock signs the TXOutput with the given address
func (out *TXOutput) Lock(address []byte) {
	pubKeyHash := base58.DecodeBase58(address)
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash) - 4]
	out.PubKeyHash = pubKeyHash
}

// IsLockedWithKey returns true if TXOutput is signed
// with the given public key
func (out *TXOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Compare(out.PubKeyHash, pubKeyHash) == 0
}

// NewCoinbaseTX returns new coinbase Transaction with miner's and stakeholder's outputs
func NewCoinbaseTX(minerAddr, stakeAddr, data string, satoshiIndex int) *Transaction {
	if data == "" {
		data = "some data"
	}

	txin := TXInput{
		OutTxID:   []byte{},
		OutIndex:  -1,
		PubKey: []byte(data),
	}

	minerSubsidyArr := make([]int, subsidy)
	for i := 0; i < subsidy; i++  {
		minerSubsidyArr[i] = i + satoshiIndex
	}

	stakeSubsidyArr := make([]int, subsidy)
	for i := 0; i < subsidy; i++  {
		stakeSubsidyArr[i] = i + satoshiIndex + subsidy
	}

	var txOutputs []TXOutput

	if minerAddr == stakeAddr {
		minerSubsidyArr = append(minerSubsidyArr, stakeSubsidyArr...)
		txout := NewTXOutput(minerSubsidyArr, minerAddr)
		txOutputs = []TXOutput{*txout}
	} else {
		txoutMiner := NewTXOutput(minerSubsidyArr, minerAddr)
		txoutStake := NewTXOutput(stakeSubsidyArr, stakeAddr)
		txOutputs = []TXOutput{*txoutMiner, *txoutStake}
	}

	tx := Transaction{
		Vin:  []TXInput{txin},
		Vout: txOutputs,
	}
	tx.ID = tx.Hash()

	return &tx
}

// NewTransaction returns new Transaction
func NewTransaction(from, to string, amount int, bc *Blockchain) (*Transaction, error) {
	var inputs []TXInput
	var outputs []TXOutput

	wallets, err := walletpkg.NewWallets(bc.AddrFile, bc.WalletFile)
	if err != nil {
		log.Panic(err)
	}
	wallet, err := wallets.GetWallet(from)
	if err != nil {
		return nil, fmt.Errorf("Failed to get wallet: %v ", err)
	}
	pubKeyHash := base58.HashPubKey(wallet.PublicKey)

	acc, validOutputs := bc.FindSpendableOutputs(pubKeyHash, amount)

	if len(acc) < amount {
		return nil, fmt.Errorf("Not enough funds ")
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

	outputs = append(outputs, *NewTXOutput(acc[:amount], to))
	if len(acc) > amount {
		outputs = append(outputs, *NewTXOutput(acc[amount:], from))
	}

	tx := Transaction{Vin: inputs, Vout: outputs}
	tx.ID = tx.Hash()

	bc.SignTransaction(&tx, wallet.PrivateKey)

	return &tx, nil
}

// Sign signs Transaction with given ecdsa.PrivateKey
func (tx *Transaction) Sign(privKey ecdsa.PrivateKey, prevTXs map[string]Transaction) {
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

		dataToSign := fmt.Sprintf("%x\n", txCopy)

		r, s, err := ecdsa.Sign(rand.Reader, &privKey, []byte(dataToSign))
		if err != nil {
			log.Panic(err)
		}
		signature := append(r.Bytes(), s.Bytes()...)

		tx.Vin[inID].Signature = signature
		txCopy.Vin[inID].PubKey = nil
	}
}

// Verify returns true if Transaction is valid
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

		dataToVerify := fmt.Sprintf("%x\n", txCopy)

		rawPubKey := ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}
		if ecdsa.Verify(&rawPubKey, []byte(dataToVerify), &r, &s) == false {
			return false
		}
		txCopy.Vin[inID].PubKey = nil
	}

	return true
}

// TrimmedCopy returns copy of Transaction
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

// Serialize serializes Transaction into bytes
func (tx Transaction) Serialize() []byte {
	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	if err != nil {
		log.Panic(err)
	}

	return encoded.Bytes()
}

// Hash returns sum256 hash of Transaction
func (tx *Transaction) Hash() []byte {
	var hash [32]byte

	txCopy := *tx
	txCopy.ID = []byte{}

	hash = sha256.Sum256(txCopy.Serialize())

	return hash[:]
}

// DeserializeTransaction deserializes Transaction from bytes
func DeserializeTransaction(data []byte) Transaction {
	var transaction Transaction

	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&transaction)
	if err != nil {
		log.Panic(err)
	}

	return transaction
}
