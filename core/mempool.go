package core

import (
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// MaxMempoolSize is the maximum number of transactions in mempool
const MaxMempoolSize = 1000

// Mempool represents a pool of pending transactions
type Mempool struct {
	mu       sync.RWMutex
	txs      map[string]*Transaction
	bySender map[string][]string // address -> tx IDs
}

// NewMempool creates a new mempool
func NewMempool() *Mempool {
	return &Mempool{
		txs:      make(map[string]*Transaction),
		bySender: make(map[string][]string),
	}
}

// Add adds a transaction to the mempool
// Returns error if transaction is invalid or mempool is full
func (m *Mempool) Add(tx *Transaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check mempool size
	if len(m.txs) >= MaxMempoolSize {
		return fmt.Errorf("mempool is full")
	}

	// Check if transaction already exists
	txID := hex.EncodeToString(tx.ID)
	if _, exists := m.txs[txID]; exists {
		return fmt.Errorf("transaction already in mempool")
	}

	// Validate transaction before adding
	if err := m.validateTx(tx); err != nil {
		return fmt.Errorf("invalid transaction: %w", err)
	}

	// Add to mempool
	m.txs[txID] = tx

	// Index by sender - derive from inputs (the spending party)
	// For coinbase transactions, there's no real sender, so skip indexing
	if !tx.IsCoinbase() && len(tx.Inputs) > 0 {
		// Use the first input's PubKey as sender identifier
		sender := hex.EncodeToString(tx.Inputs[0].PubKey)
		m.bySender[sender] = append(m.bySender[sender], txID)
	}

	return nil
}

// Remove removes a transaction from the mempool
func (m *Mempool) Remove(txID []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := hex.EncodeToString(txID)
	tx, exists := m.txs[key]
	if !exists {
		return
	}

	// Remove from sender index
	if !tx.IsCoinbase() && len(tx.Inputs) > 0 {
		sender := hex.EncodeToString(tx.Inputs[0].PubKey)
		if ids, ok := m.bySender[sender]; ok {
			for i, id := range ids {
				if id == key {
					m.bySender[sender] = append(ids[:i], ids[i+1:]...)
					break
				}
			}
		}
	}

	delete(m.txs, key)
}

// Get retrieves a transaction by ID
func (m *Mempool) Get(txID []byte) (*Transaction, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tx, ok := m.txs[hex.EncodeToString(txID)]
	return tx, ok
}

// GetBySender returns all transactions from a sender
func (m *Mempool) GetBySender(address []byte) []*Transaction {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*Transaction
	sender := hex.EncodeToString(address)
	txIDs := m.bySender[sender]
	for _, txID := range txIDs {
		if tx, ok := m.txs[txID]; ok {
			result = append(result, tx)
		}
	}
	return result
}

// List returns all transactions in mempool
func (m *Mempool) List() []*Transaction {
	m.mu.RLock()
	defer m.mu.RUnlock()

	txs := make([]*Transaction, 0, len(m.txs))
	for _, tx := range m.txs {
		txs = append(txs, tx)
	}
	return txs
}

// Size returns the number of transactions in mempool
func (m *Mempool) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.txs)
}

// Clear removes all transactions from mempool
func (m *Mempool) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.txs = make(map[string]*Transaction)
	m.bySender = make(map[string][]string)
}

// Contains checks if a transaction is in mempool
func (m *Mempool) Contains(txID []byte) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.txs[hex.EncodeToString(txID)]
	return exists
}

// validateTx performs basic transaction validation before adding to mempool
func (m *Mempool) validateTx(tx *Transaction) error {
	// Check for nil
	if tx == nil {
		return fmt.Errorf("nil transaction")
	}

	// Check ID exists
	if len(tx.ID) == 0 {
		return fmt.Errorf("transaction has no ID")
	}

	// Check inputs
	if len(tx.Inputs) == 0 {
		return fmt.Errorf("transaction has no inputs")
	}

	// Check outputs
	if len(tx.Outputs) == 0 {
		return fmt.Errorf("transaction has no outputs")
	}

	// Check output values are positive
	for _, out := range tx.Outputs {
		if out.Value <= 0 {
			return fmt.Errorf("output value must be positive")
		}
	}

	// Verify signature for non-coinbase
	if !tx.IsCoinbase() {
		if !tx.Verify() {
			return fmt.Errorf("invalid signature")
		}
	}

	return nil
}

// RemoveExpired removes transactions older than maxAge
func (m *Mempool) RemoveExpired(maxAge time.Duration) {
	// This would need timestamp tracking - simplified for now
	// In production, you'd store createdAt for each tx
}

// GetConflicts returns transactions that conflict with the given input
func (m *Mempool) GetConflicts(tx *Transaction) []*Transaction {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var conflicts []*Transaction

	for _, existing := range m.txs {
		for _, in := range tx.Inputs {
			for _, existingIn := range existing.Inputs {
				if in.Txid != nil && existingIn.Txid != nil {
					if string(in.Txid) == string(existingIn.Txid) && in.OutIndex == existingIn.OutIndex {
						conflicts = append(conflicts, existing)
					}
				}
			}
		}
	}

	return conflicts
}