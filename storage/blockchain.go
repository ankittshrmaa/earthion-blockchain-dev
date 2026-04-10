package storage

import (
	"encoding/gob"
	"os"

	"earthion/core"
)

const ChainFile = "blockchain.dat"

// SaveBlockchain persists blockchain to file
func SaveBlockchain(bc *core.Blockchain, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Only serialize the Blocks array (not unexported fields)
	encoder := gob.NewEncoder(file)
	return encoder.Encode(bc.Blocks)
}

// LoadBlockchain reads blockchain from file
func LoadBlockchain(filename string) (*core.Blockchain, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, err
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Create new blockchain and load data
	bc := core.NewBlockchain()
	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&bc.Blocks)
	if err != nil {
		return nil, err
	}

	// Initialize new fields (not serialized by gob)
	bc.SetFilename(filename)
	
	return bc, nil
}
