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

	encoder := gob.NewEncoder(file)
	return encoder.Encode(bc)
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

	bc := &core.Blockchain{}
	decoder := gob.NewDecoder(file)
	err = decoder.Decode(bc)
	if err != nil {
		return nil, err
	}

	return bc, nil
}
