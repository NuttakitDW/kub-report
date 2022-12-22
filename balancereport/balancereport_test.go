package balancereport

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	ctx = context.Background()
	client,_ = ethclient.Dial("https://mainnet.infura.io/v3/8e832100bb9d451887b920a0935dc120")
	br,_ = New(client)
)

func TestGetBalanceOneAccount(t *testing.T) {
	balance, err := br.getBalance(ctx, "0xBE0eB53F46cd790Cd13851d5EFf43D12404d33E8", 10120447)
	if err != nil {
		t.Fatal(err)
	}

	if balance.String() != "2507764461397104375785566" {
		t.Error("Expected balance to be 2507764461397104375785566, but got ", balance.String())
	}
}

func TestGetBlockRange(t *testing.T) {
	blocks, err := br.getBlockRange(ctx, 1671469200, 1671555600)
	if err != nil {
		t.Fatal(err)
	}
	if blocks == nil {
		t.Error("Expected blocks have member more that 0, but got ", len(blocks))
	}
}

func TestGetDailyChg(t *testing.T) {

}


