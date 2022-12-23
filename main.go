package main

import (
	"context"
	"fmt"
	"kub-report/balancereport"
	"github.com/ethereum/go-ethereum/ethclient"
	"time"
)

func main() {
	// Connect to the Ethereum mainnet using the Infura RPC URL
	client, err := ethclient.Dial("https://mainnet.infura.io/v3/8e832100bb9d451887b920a0935dc120")
	if err != nil {
		fmt.Printf("Error connecting to Ethereum mainnet: %v\n", err)
		return
	}

	// Create a new balance report instance
	br, err := balancereport.New(client)
	if err != nil {
		fmt.Printf("Error creating balance report: %v\n", err)
		return
	}

	loc, _ := time.LoadLocation("Asia/Bangkok")
	startTimestamp := time.Date(2022, 11, 1, 0, 0, 0, 0, loc).Unix()
	endTimestamp := time.Date(2022, 12, 1, 0, 0, 0, 0, loc).Unix()

	// Set the addresses to include in the report
	addresses := []string{
		"0xddbd2b932c763ba5b1b7ae3b362eac3e8d40121a",
		"0x742d35cc6634c0532925a3b844bc454e4438f44e",
		"0xc94770007dda54cF92009BFF0dE90c06F603a09f",
	}

	// Generate the balance report
	if err := br.GetReport(context.Background(), startTimestamp, endTimestamp, addresses); err != nil {
		fmt.Printf("Error generating balance report: %v\n", err)
		return
	}

	fmt.Println("Balance report generated successfully!")
}
