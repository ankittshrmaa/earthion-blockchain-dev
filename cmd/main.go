package main

import (
	"fmt"
	"os"

	"earthion/cli"
	"earthion/core"
	"earthion/storage"
	"earthion/wallet"
)

const (
	WalletFile = "wallet.dat"
	ChainFile  = "blockchain.dat"
)

func main() {
	fmt.Println("=== Earthion ===")
	fmt.Println("Loading wallet...")

	// Load or create wallet
	wal := loadOrCreateWallet()
	fmt.Printf("Wallet address: %s\n\n", wal.AddressHex())

	// Load or create blockchain
	bc := loadOrCreateBlockchain()
	fmt.Printf("Chain loaded: %d blocks\n\n", len(bc.Blocks))

	// Run CLI
	c := cli.NewCLI(bc, wal)
	c.Run()

	// Save on exit
	fmt.Println("\nSaving blockchain...")
	if err := storage.SaveBlockchain(bc, ChainFile); err != nil {
		fmt.Printf("Error saving chain: %v\n", err)
	} else {
		fmt.Println("Blockchain saved!")
	}
}

func loadOrCreateWallet() *wallet.Wallet {
	wal, err := wallet.LoadWallet(WalletFile)
	if err != nil {
		wal, err = wallet.NewWallet()
		if err != nil {
			fmt.Printf("Error creating wallet: %v\n", err)
			os.Exit(1)
		}
		if err := wal.Save(WalletFile); err != nil {
			fmt.Printf("Error saving wallet: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("New wallet created!")
	}
	return wal
}

func loadOrCreateBlockchain() *core.Blockchain {
	bc, err := storage.LoadBlockchain(ChainFile)
	if err != nil {
		if os.IsNotExist(err) {
			bc = core.NewBlockchain()
			fmt.Println("New blockchain created!")
		} else {
			fmt.Printf("Error loading chain: %v, creating new...\n", err)
			bc = core.NewBlockchain()
		}
	}
	return bc
}
