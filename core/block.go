package core

import (
	"bytes"
	"encoding/gob"
	"time"

	"earthion/crypto"
)

type Block struct {
	Index        int
	Timestamp    int64
	PrevHash     []byte
	MerkleRoot   []byte
	Hash         []byte
	Nonce        int
	Transactions []*Transaction
}

// Serialize block (used for storage / hashing if needed)
func (b *Block) Serialize() []byte {
	var res bytes.Buffer
	encoder := gob.NewEncoder(&res)
	_ = encoder.Encode(b)
	return res.Bytes()
}

// Create new block with transactions
func NewBlock(txs []*Transaction, prevHash []byte, index int) *Block {
	// Build transaction hashes
	txHashes := make([][]byte, len(txs))
	for i, tx := range txs {
		txHashes[i] = tx.ID
	}

	// Calculate Merkle root
	var merkleRoot []byte
	if len(txHashes) > 0 {
		tree := crypto.NewMerkleTree(txHashes)
		merkleRoot = tree.RootHash()
	} else {
		// Empty block - use double hash of empty bytes
		merkleRoot = crypto.DoubleHash([]byte{})
	}

	block := &Block{
		Index:        index,
		Timestamp:    time.Now().Unix(),
		PrevHash:     prevHash,
		MerkleRoot:   merkleRoot,
		Transactions: txs,
	}

	pow := NewProofOfWork(block)
	nonce, hash := pow.Run()

	block.Nonce = nonce
	block.Hash = hash

	return block
}

// First block in chain
func GenesisBlock() *Block {
	return NewBlock([]*Transaction{}, []byte{}, 0)
}