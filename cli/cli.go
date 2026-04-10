package cli

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"

	"earthion/core"
	"earthion/wallet"
)

const WalletFile = "wallet.dat"

type CLI struct {
	bc  *core.Blockchain
	wal *wallet.Wallet
	sc  *bufio.Scanner
}

func NewCLI(bc *core.Blockchain, wal *wallet.Wallet) *CLI {
	return &CLI{
		bc:  bc,
		wal: wal,
		sc:  bufio.NewScanner(os.Stdin),
	}
}

func (c *CLI) prompt() string {
	fmt.Print("> ")
	c.sc.Scan()
	return c.sc.Text()
}

func (c *CLI) Run() {
	fmt.Println("=== Earthion CLI ===")
	fmt.Println("Commands: balance, send, mine, validate, list, help, exit")
	fmt.Printf("Current wallet: %s\n\n", c.wal.AddressHex())

	for {
		input := c.prompt()
		args := strings.Fields(input)
		if len(args) == 0 {
			continue
		}

		cmd := args[0]
		switch cmd {
		case "balance":
			c.balance()
		case "send":
			if len(args) < 3 {
				fmt.Println("Usage: send <to_address> <amount>")
				continue
			}
			c.send(args[1], args[2])
		case "mine":
			c.mine()
		case "validate":
			c.validate()
		case "list":
			c.listBlocks()
		case "help":
			fmt.Println("Commands: balance, send, mine, validate, list, help, exit")
		case "exit":
			fmt.Println("Goodbye!")
			return
		default:
			fmt.Println("Unknown command. Type 'help' for available commands.")
		}
	}
}

func (c *CLI) balance() {
	// Use raw address (20-byte pubkey hash) for balance lookup
	balance := c.bc.GetBalance(c.wal.GetRawAddress())
	fmt.Printf("Balance: %d\n", balance)
}

func (c *CLI) send(toHex, amountStr string) {
	toAddr, err := hex.DecodeString(toHex)
	if err != nil {
		fmt.Println("Invalid address")
		return
	}

	amount, err := strconv.Atoi(amountStr)
	if err != nil {
		fmt.Println("Invalid amount")
		return
	}

	tx, err := core.NewTransaction(c.wal, toAddr, amount, c.bc)
	if err != nil {
		fmt.Printf("Error creating transaction: %v\n", err)
		return
	}
	c.bc.AddBlock([]*core.Transaction{tx})

	txID := hex.EncodeToString(tx.ID)[:16]
	fmt.Printf("Sent %d to %s\n", amount, toHex[:16])
	fmt.Printf("Transaction ID: %s\n", txID)
}

func (c *CLI) mine() {
	// Coinbase transaction with reward
	reward := 50
	// Use raw pubkey for coinbase output
	coinbase := core.CoinbaseTx(c.wal.PublicKey, reward)
	c.bc.AddBlock([]*core.Transaction{coinbase})

	fmt.Printf("Mined new block! Reward: %d\n", reward)
	// Use raw address for balance lookup
	fmt.Printf("New balance: %d\n", c.bc.GetBalance(c.wal.GetRawAddress()))
}

func (c *CLI) validate() {
	if c.bc.Validate() {
		fmt.Println("✓ Chain is valid!")
	} else {
		fmt.Println("✗ Chain is INVALID!")
	}
}

func (c *CLI) listBlocks() {
	for i, block := range c.bc.Blocks {
		fmt.Printf("\nBlock %d:\n", i)
		fmt.Printf("  Index: %d\n", block.Index)
		fmt.Printf("  Hash: %s\n", hex.EncodeToString(block.Hash[:16]))
		fmt.Printf("  PrevHash: %s\n", hex.EncodeToString(block.PrevHash[:16]))
		fmt.Printf("  Transactions: %d\n", len(block.Transactions))
	}
}
