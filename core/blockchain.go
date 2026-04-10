package core

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"

	"earthion/crypto"
)

type Blockchain struct {
	Blocks []*Block
}

func NewBlockchain() *Blockchain {
	return &Blockchain{[]*Block{GenesisBlock()}}
}

func (bc *Blockchain) AddBlock(txs []*Transaction) {
	prev := bc.Blocks[len(bc.Blocks)-1]
	newBlock := NewBlock(txs, prev.Hash, prev.Index+1)
	bc.Blocks = append(bc.Blocks, newBlock)
}

// Validate checks the entire blockchain for integrity
func (bc *Blockchain) Validate() bool {
	if len(bc.Blocks) == 0 {
		return false
	}

	genesis := bc.Blocks[0]
	if !bytes.Equal(genesis.PrevHash, []byte{}) {
		log.Println("Genesis block has incorrect prevHash")
		return false
	}

	// Track spent outputs to detect double-spends
	spentOutputs := make(map[string]bool)

	for i := 1; i < len(bc.Blocks); i++ {
		block := bc.Blocks[i]
		prevBlock := bc.Blocks[i-1]

		if !bytes.Equal(block.PrevHash, prevBlock.Hash) {
			log.Printf("Block %d: prevHash doesn't match previous block hash\n", i)
			return false
		}

		pow := NewProofOfWork(block)
		if !pow.Validate() {
			log.Printf("Block %d: PoW validation failed\n", i)
			return false
		}

		// Validate Merkle root matches transactions
		txHashes := make([][]byte, len(block.Transactions))
		for j, tx := range block.Transactions {
			txHashes[j] = tx.ID
		}
		merkleTree := crypto.NewMerkleTree(txHashes)
		if !bytes.Equal(block.MerkleRoot, merkleTree.RootHash()) {
			log.Printf("Block %d: Merkle root mismatch\n", i)
			return false
		}

		for j, tx := range block.Transactions {
			if tx.IsCoinbase() {
				continue
			}

			// Check for double-spend within this transaction
			inputsSeen := make(map[string]bool)
			for _, in := range tx.Inputs {
				if in.OutIndex == -1 {
					continue
				}
				inTxID := hex.EncodeToString(in.Txid)
				key := fmt.Sprintf("%s:%d", inTxID, in.OutIndex)
				
				if inputsSeen[key] {
					log.Printf("Block %d: Transaction %d has double-spend input\n", i, j)
					return false
				}
				inputsSeen[key] = true

				// Check against previously spent outputs in the chain
				if spentOutputs[key] {
					log.Printf("Block %d: Transaction %d attempts to spend already-spent output\n", i, j)
					return false
				}
				spentOutputs[key] = true
			}

			if !tx.Verify() {
				log.Printf("Block %d: Transaction %d verification failed\n", i, j)
				return false
			}
		}
	}

	return true
}

func (bc *Blockchain) GetBlock(blockHash []byte) *Block {
	for _, block := range bc.Blocks {
		if bytes.Equal(block.Hash, blockHash) {
			return block
		}
	}
	return nil
}

func (bc *Blockchain) FindTransaction(txID []byte) *Transaction {
	for _, block := range bc.Blocks {
		for _, tx := range block.Transactions {
			if bytes.Equal(tx.ID, txID) {
				return tx
			}
		}
	}
	return nil
}

// UTXOIndex returns a map of unspent transaction outputs
// Key: "txID:outIdx", Value: TXOutput
func (bc *Blockchain) UTXOIndex() map[string]TXOutput {
	utxos := make(map[string]TXOutput)

	for _, block := range bc.Blocks {
		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)

			for outIdx, out := range tx.Outputs {
				key := fmt.Sprintf("%s:%d", txID, outIdx)
				utxos[key] = out
			}

			// Remove spent inputs - but first verify the UTXO exists
			// This prevents double-spend attacks where same output is spent multiple times
			for _, in := range tx.Inputs {
				if in.OutIndex == -1 {
					continue // coinbase has no valid input
				}
				inTxID := hex.EncodeToString(in.Txid)
				key := fmt.Sprintf("%s:%d", inTxID, in.OutIndex)
				
				// Only delete if it exists - this prevents double-spend exploits
				// If the key doesn't exist, it means this input was already spent
				// (or references a non-existent output), which indicates an invalid chain
				if _, exists := utxos[key]; exists {
					delete(utxos, key)
				}
			}
		}
	}

	return utxos
}

// GetBalance returns the balance for a given address
// Note: address should be the raw 20-byte pubkey hash (not Base58 encoded)
func (bc *Blockchain) GetBalance(address []byte) int {
	utxos := bc.UTXOIndex()
	balance := 0

	for _, out := range utxos {
		// Compare against the pubkey hash stored in the output
		outPubKeyHash := crypto.Hash(out.PubKey)[:20]
		if bytes.Equal(outPubKeyHash, address) {
			balance += out.Value
		}
	}

	return balance
}
