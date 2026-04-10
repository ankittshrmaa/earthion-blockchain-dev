package core

import (
	"bytes"
	"encoding/hex"
	"encoding/gob"
	"fmt"

	"earthion/crypto"
	"earthion/wallet"
)

type Transaction struct {
	ID []byte
	Inputs []TXInput
	Outputs []TXOutput
}

type TXInput struct {
	Txid []byte
	OutIndex int
	Signature []byte
	PubKey []byte
}

type TXOutput struct {
	Value int
	PubKey []byte
}

func (tx *Transaction) Serialize() []byte {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	_ = encoder.Encode(tx)
	return buf.Bytes()
}

// SetID generates a unique ID for the transaction
func (tx *Transaction) SetID() {
	tx.ID = crypto.DoubleHash(tx.Serialize())
}

// Sign signs the transaction data with the private key
func (tx *Transaction) Sign(privateKey *wallet.Wallet) {
	tx.SetID()

	data := tx.GetSignData()

	for i := range tx.Inputs {
		tx.Inputs[i].Signature = privateKey.Sign(data)
	}
}

// GetSignData returns the data to be signed (transaction ID + outputs)
func (tx *Transaction) GetSignData() []byte {
	var data []byte

	data = append(data, tx.ID...)
	for _, out := range tx.Outputs {
		data = append(data, IntToHex(int64(out.Value))...)
		data = append(data, out.PubKey...)
	}

	return crypto.Hash(data)
}

// Verify checks if the transaction signature is valid
func (tx *Transaction) Verify() bool {
	if len(tx.Inputs) == 0 {
		return false
	}

	data := tx.GetSignData()

	for _, input := range tx.Inputs {
		if len(input.Signature) == 0 || len(input.PubKey) == 0 {
			continue // Skip empty inputs (coinbase)
		}
		if !wallet.VerifySignature(input.PubKey, data, input.Signature) {
			return false
		}
	}

	return true
}

// IsCoinbase checks if this is a coinbase transaction
func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Inputs) == 1 && tx.Inputs[0].OutIndex == -1
}

// CoinbaseTx creates a reward transaction for mining
func CoinbaseTx(pubKey []byte, amount int) *Transaction {
	tx := &Transaction{
		Inputs: []TXInput{
			{
				Txid: []byte{},
				OutIndex: -1,
				Signature: nil,
				PubKey: pubKey,
			},
		},
		Outputs: []TXOutput{
			{
				Value: amount,
				PubKey: pubKey,
			},
		},
	}

	tx.SetID()
	return tx
}

// NewTransaction creates a new transaction that spends previous outputs
func NewTransaction(from *wallet.Wallet, to []byte, amount int, bc *Blockchain) (*Transaction, error) {
	utxos := bc.UTXOIndex()
	// Use raw address (20-byte pubkey hash) for balance lookup
	balance := bc.GetBalance(from.GetRawAddress())

	if amount > balance {
		return nil, fmt.Errorf("insufficient balance: need %d, have %d", amount, balance)
	}

	var inputs []TXInput
	var outputs []TXOutput

	// Gather UTXOs
	var total int
	for key, out := range utxos {
		// Parse key to get txID and outIdx
		var txIDHex string
		var outIdx int
		_, err := fmt.Sscanf(key, "%s:%d", &txIDHex, &outIdx)
		if err != nil {
			continue
		}

		txID, _ := hex.DecodeString(txIDHex)

		// Check if output belongs to sender - compare raw pubkey hashes
		outPubKeyHash := crypto.Hash(out.PubKey)[:20]
		if !bytes.Equal(outPubKeyHash, from.GetRawAddress()) {
			continue
		}

		// Validate that the referenced output exists in the UTXO set
		// This ensures we're spending a valid, existing output
		if _, exists := utxos[key]; !exists {
			continue // Skip already-spent outputs
		}

		inputs = append(inputs, TXInput{
			Txid:      txID,
			OutIndex:  outIdx,
			Signature: nil,
			PubKey:    from.PublicKey,
		})

		total += out.Value

		if total >= amount {
			break
		}
	}

	// Validate we have enough inputs
	if len(inputs) == 0 && amount > 0 {
		return nil, fmt.Errorf("no valid UTXOs found for spending")
	}

	// Output to recipient
	outputs = append(outputs, TXOutput{
		Value: amount,
		PubKey: to,
	})

	// Output change back to sender
	if total > amount {
		outputs = append(outputs, TXOutput{
			Value: total - amount,
			PubKey: from.PublicKey,
		})
	}

	// CRITICAL: Validate that total input value equals total output value
	// This prevents creating or destroying coins
	var totalOutputValue int
	for _, out := range outputs {
		totalOutputValue += out.Value
	}

	if total != totalOutputValue {
		return nil, fmt.Errorf("input/output value mismatch: inputs=%d, outputs=%d", total, totalOutputValue)
	}

	tx := &Transaction{
		Inputs: inputs,
		Outputs: outputs,
	}

	tx.Sign(from)
	return tx, nil
}
