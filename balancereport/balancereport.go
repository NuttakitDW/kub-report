package balancereport

import (
	"fmt"
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
	// Initialize the savedBalance and isSaved maps

	br := &BalanceReport{
		client:       client,
		goblock:      gb,
	}
	return br, nil
}

func (br *BalanceReport) getBlockRange(ctx context.Context, start int64, end int64) ([]int64, error) {
	fmt.Println("Getting block range...")
	defer fmt.Println("Finished getting block range.")

	day := 24 * time.Hour
	blocks, err := br.goblock.GetEvery(ctx, day, start, end)
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
	br.isSaved = make(map[string]map[int64]bool)
	br.savedBalance = make(map[string]map[int64]*big.Int)
	br.isSaved[address] = make(map[int64]bool)
	br.savedBalance[address] = make(map[int64]*big.Int)
	br.isSaved[address][block] = true
	br.savedBalance[address][block] = balance
	return balance, nil
}

func (br *BalanceReport) GetReport(ctx context.Context, start int64, end int64, addresses []string) error {
	blockRange, err := br.getBlockRange(ctx, start, end)
	if err != nil {
		return err
	}

	// Generate the CSV filename using the current date and time
	csvFilename := fmt.Sprintf("balancereport[%s].csv", time.Now())
	file, err := os.Create(csvFilename)
	if err != nil {
		return err
	}
	defer file.Close()

	w := csv.NewWriter(file)
	defer w.Flush()

	// Write the CSV header row
	headerRow := []string{"date", "timestamp", "block", "address", "daily chg", "ending balance"}
	fmt.Println(headerRow)
	if err := w.Write(headerRow); err != nil {
		return err
	}

	// Initialize the current balance mapping
	currentBalances := make(map[string]*big.Int)
	for _, address := range addresses {
		currentBalances[address] = big.NewInt(0)
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

			dailyChg := new(big.Int).Sub(balance, currentBalances[address])
			currentBalances[address] = balance

			timestamp := big.NewInt(block).Int64()
			// Use the "Jan-02-2006" layout to format the date
			date := time.Unix(timestamp, 0).Format("Jan-02-2006")

			row := []string{date, strconv.FormatInt(timestamp, 10), strconv.FormatInt(block, 10), address, balance.String(), dailyChg.String()}
			fmt.Println(row)
			if err := w.Write(row); err != nil {
				return err
			}
		}
	}

	return nil
}