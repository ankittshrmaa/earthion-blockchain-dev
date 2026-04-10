package core

import (
	"bytes"
	"math"
	"math/big"

	"earthion/crypto"
)

const Difficulty = 18

type ProofOfWork struct {
	Block  *Block
	Target *big.Int
}

func NewProofOfWork(b *Block) *ProofOfWork {
	target := big.NewInt(1)
	target.Lsh(target, 256-Difficulty)
	return &ProofOfWork{b, target}
}

func (pow *ProofOfWork) prepareData(nonce int) []byte {
	txData := []byte{}

	for _, tx := range pow.Block.Transactions {
		txData = append(txData, tx.Serialize()...)
	}

	return bytes.Join(
		[][]byte{
			IntToHex(int64(pow.Block.Index)),
			IntToHex(pow.Block.Timestamp),
			pow.Block.PrevHash,
			pow.Block.MerkleRoot,
			txData,
			IntToHex(int64(nonce)),
		},
		[]byte{},
	)
}

func (pow *ProofOfWork) Run() (int, []byte) {
	var hashInt big.Int
	var hash []byte
	nonce := 0

	for nonce < math.MaxInt64 {
		data := pow.prepareData(nonce)
		hash = crypto.DoubleHash(data)

		hashInt.SetBytes(hash)

		if hashInt.Cmp(pow.Target) == -1 {
			break
		}
		nonce++
	}

	return nonce, hash
}

// Validate verifies the proof of work
func (pow *ProofOfWork) Validate() bool {
	data := pow.prepareData(pow.Block.Nonce)
	hash := crypto.DoubleHash(data)

	var hashInt big.Int
	hashInt.SetBytes(hash)

	return hashInt.Cmp(pow.Target) == -1
}