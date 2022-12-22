package balancereport

import (
	"context"
	"time"
	"math/big"
	"kub-report/goblock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"encoding/csv"
	"os"
	"strconv"
)

// BalanceReport represents the structure of a balance report.
type BalanceReport struct {
	client            *ethclient.Client
	goblock           *goblock.GoBlock
	savedBalance 	  map[string]map[int64]*big.Int
	isSaved 		  map[string]map[int64]bool
}

// New creates a new transfer report with the given Ethereum client and options.
func New(client *ethclient.Client) (*BalanceReport, error) {
	gb, err := goblock.New(client)
	if err != nil {
		return nil, err
	}
	tr := &BalanceReport{
		client:    client,
		goblock:   gb,
	}
	return tr, nil
}

func (br *BalanceReport) getBlockRange(ctx context.Context, start int64, end int64) ([]int64, error) {
	day := 24 * time.Hour
	blocks, err := br.goblock.GetEvery(ctx, day, start - int64(day), end)
	if err != nil {
		return []int64{}, err
	}
	return blocks, nil
}

func (br *BalanceReport) getBalance(ctx context.Context, address string, block int64) (*big.Int, error) {
	if br.isSaved[address][block] {
		return br.savedBalance[address][block], nil
	}
	account := common.HexToAddress(address)
	blockNumber := big.NewInt(block)
	balance, err := br.client.BalanceAt(ctx, account, blockNumber)
	if err != nil {
		return big.NewInt(0), err
	}
	br.savedBalance[address][block] = balance
	return balance, nil
}

func (br *BalanceReport) GetReport(ctx context.Context, start int64, end int64, addresses []string) error {
	blockRange, err := br.getBlockRange(ctx, start, end)
	if err != nil {
		return err
	}

	file, err := os.Create("balance_report.csv")
	if err != nil {
		return err
	}
	defer file.Close()

	w := csv.NewWriter(file)
	defer w.Flush()

	// Write the CSV header row
	headerRow := []string{"date", "timestamp", "block", "address", "daily chg", "ending balance"}
	if err := w.Write(headerRow); err != nil {
		return err
	}

	for i, block := range blockRange {
		// Skip the first block since we don't have a previous balance to compare against
		if i == 0 {
			continue
		}

		for _, address := range addresses {
			balance, err := br.getBalance(ctx, address, block)
			if err != nil {
				return err
			}

			prevBalance, err := br.getBalance(ctx, address, blockRange[i-1])
			if err != nil {
				return err
			}

			dailyChg := new(big.Int).Sub(balance, prevBalance)

			timestamp := big.NewInt(block).Int64()
			date := time.Unix(timestamp, 0).Format("2006-01-02")

			row := []string{date, strconv.FormatInt(timestamp, 10), strconv.FormatInt(block, 10), address, balance.String(), dailyChg.String()}
			if err := w.Write(row); err != nil {
				return err
			}
		}
	}

	return nil
}
