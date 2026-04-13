package core

import (
	"bytes"
	"math"
	"math/big"

	"earthion/crypto"
)

const (
	InitialDifficulty    = 18
	TargetBlockTime      = 10 // seconds - desired block time
	AdjustmentInterval   = 10 // adjust difficulty every N blocks
	MinDifficulty        = 1  // Minimum difficulty (easiest allowed)
	MaxDifficulty        = 32 // Maximum difficulty (hardest allowed, prevents overflow)
)

type ProofOfWork struct {
	Block  *Block
	Target *big.Int
}

// CurrentDifficulty returns the current difficulty for mining a new block
func CurrentDifficulty(chain []*Block) uint32 {
	if len(chain) < AdjustmentInterval+1 {
		return InitialDifficulty
	}

	// Calculate time taken for last N blocks
	oldBlock := chain[len(chain)-AdjustmentInterval]
	newBlock := chain[len(chain)-1]
	timeDiff := newBlock.Timestamp - oldBlock.Timestamp

	// Expected time = N blocks * target block time
	expectedTime := int64(AdjustmentInterval * TargetBlockTime)

	// If blocks are coming too fast, increase difficulty
	// If blocks are too slow, decrease difficulty
	ratio := float64(expectedTime) / float64(timeDiff)

	currentDiff := chain[len(chain)-1].Difficulty

	var newDiff uint32
	if ratio < 1.0 {
		// Blocks too fast - increase difficulty
		newDiff = currentDiff + 1
	} else if ratio > 2.0 {
		// Blocks very slow - decrease difficulty
		if currentDiff >= 3 {
			newDiff = currentDiff - 2
		} else if currentDiff >= 2 {
			newDiff = currentDiff - 1
		} else {
			newDiff = currentDiff
		}
	} else if ratio > 1.0 {
		// Blocks slightly slow - decrease slightly
		if currentDiff >= 2 {
			newDiff = currentDiff - 1
		} else {
			newDiff = currentDiff
		}
	} else {
		newDiff = currentDiff
	}

	// Enforce difficulty bounds
	if newDiff < MinDifficulty {
		return MinDifficulty
	}
	if newDiff > MaxDifficulty {
		return MaxDifficulty
	}

	return newDiff
}

func NewProofOfWork(b *Block) *ProofOfWork {
	difficulty := b.Difficulty
	if difficulty == 0 {
		difficulty = InitialDifficulty
	}
	// Enforce bounds in case loaded from storage with invalid value
	if difficulty < MinDifficulty {
		difficulty = MinDifficulty
	}
	if difficulty > MaxDifficulty {
		difficulty = MaxDifficulty
	}

	target := big.NewInt(1)
	target.Lsh(target, 256-uint(difficulty))
	return &ProofOfWork{b, target}
}

// prepareData creates the data to be hashed for PoW
// Includes all critical block fields for mining security
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
			IntToHex(int64(pow.Block.Difficulty)),
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

	// Verify the calculated hash matches the stored block hash
	if !bytes.Equal(hash, pow.Block.Hash) {
		return false
	}

	// Also verify hash meets difficulty target
	var hashInt big.Int
	hashInt.SetBytes(hash)

	return hashInt.Cmp(pow.Target) == -1
}