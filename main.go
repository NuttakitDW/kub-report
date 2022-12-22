package main

import (
	"fmt"
	"context"
	"log"
	"math/big"
	
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	ctx := context.Background()
	client, err := ethclient.Dial("https://mainnet.infura.io/v3/8e832100bb9d451887b920a0935dc120")
	if err != nil {
		log.Fatal(err)
	}
	account := common.HexToAddress("0xBE0eB53F46cd790Cd13851d5EFf43D12404d33E8")
	blockNumber := big.NewInt(16241410)
	balance, err := client.BalanceAt(ctx, account, blockNumber)
	if err != nil {
	log.Fatal(err)
	}
	fmt.Println(balance)
}
