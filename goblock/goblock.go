package goblock

import (
	"context"
	"math"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"golang.org/x/exp/slices"
)

type Block struct {
	number    int64
	timestamp int64
}

type GoBlock struct {
	savedBlocks   map[int64]Block // map of saved blocks, with block number as the key
	checkedBlocks map[int64][]int64 // map of checked blocks, with timestamp as the key
	firstBlock    Block // first block
	latestBlock   Block // latest block
	blockTime     float64 // average time between blocks
	client        *ethclient.Client // Ethereum client
}

// New creates a new GoBlock instance with the given Ethereum client
func New(client *ethclient.Client) (*GoBlock, error) {
	return &GoBlock{
		savedBlocks:   make(map[int64]Block),
		checkedBlocks: make(map[int64][]int64),
		client:        client,
	}, nil
}

// GetDate returns the block number that was created at the given timestamp
func (gb *GoBlock) GetDate(ctx context.Context, date int64) (int64, error) {

	// If the first or latest block or the block time is not set, get the boundaries
	if (gb.firstBlock == Block{} || gb.latestBlock == Block{} || gb.blockTime == 0) {
		err := gb.getBoundaries(ctx)
		if err != nil {
			return 0, err
		}
	}
	// If the given timestamp is before the first block, return 1
	if date < (gb.firstBlock.timestamp) {
		return 1, nil
	}
	// If the given timestamp is after or at the same time as the latest block, return the latest block number
	if date >= (gb.latestBlock.timestamp) {
		return gb.latestBlock.number, nil
	}
	// Initialize an empty slice of block numbers for the checked blocks with the given timestamp
	gb.checkedBlocks[date] = []int64{}
	// Get the predicted block using the block time and first block as reference
	predictedBlock, err := gb.getBlockWrapper(
		ctx,
		int64((float64(date-gb.firstBlock.timestamp))/gb.blockTime),
	)
	if err != nil {
		return 0, err
	}
	// Find the block that was created closest to the given timestamp
	return gb.findBetter(ctx, date, predictedBlock, true)
}

func (gb *GoBlock) GetDateAdv(ctx context.Context, date int64, after bool, refresh bool) (int64, error) {
	// Check if the first and latest blocks, as well as the block time, have been set or if refresh is true
	// If not, retrieve the boundary blocks and block time
	if (gb.firstBlock == Block{} || gb.latestBlock == Block{} || gb.blockTime == 0 || refresh) {
		gb.getBoundaries(ctx)
	}
	
	// Check if the input date is before the first block
	if date < (gb.firstBlock.timestamp) {
		return 1, nil
	}
	
	// Check if the input date is before the latest block
	if date < (gb.latestBlock.timestamp) {
		return gb.latestBlock.number, nil
	}
	
	// Initialize the list of checked blocks for the input date
	gb.checkedBlocks[date] = []int64{}
	
	// Calculate the predicted block based on the input date and the block time
	predictedBlock, err := gb.getBlockWrapper(
		ctx,
		int64(math.Ceil(float64((date-gb.firstBlock.timestamp))/float64(gb.blockTime))),
	)
	if err != nil {
		return 0, err
	}
	
	// Find a better block using the input date, predicted block, and after flag
	return gb.findBetter(ctx, date, predictedBlock, after)
}


func (gb *GoBlock) GetEvery(ctx context.Context, duration time.Duration, start int64, end int64) ([]int64, error) {
	// Initialize a slice of dates with the start date
	var dates []time.Time
	current := time.Unix(start, 0)
	// Iterate through each date from start to end, adding the duration to the current date each iteration
	for !current.After(time.Unix(end, 0)) {
		dates = append(dates, current)
		current = current.Add(duration)
	}
	
	// Check if the first and latest blocks, as well as the block time, have been set
	// If not, retrieve the boundary blocks and block time
	if (gb.firstBlock == Block{} || gb.latestBlock == Block{} || gb.blockTime == 0) {
		gb.getBoundaries(ctx)
	}
	
	// Initialize a slice of blocks
	var blocks []int64
	
	// Iterate through each date, finding the block for that date using the GetDate function
	// If an error is returned, return nil and the error
	for _, date := range dates {
		block, err := gb.GetDate(ctx, date.Unix())
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}
	
	// Return the slice of blocks
	return blocks, nil
}

func (gb *GoBlock) findBetter(
	ctx context.Context,
	date int64,
	predictedBlock Block,
	after bool,
) (int64, error) {
	// blockTime is the time it takes to generate a block, in seconds
	blockTime := gb.blockTime

	// Check if the predicted block is a better block than the current block
	isBetterBlock, err := gb.isBetterBlock(ctx, date, predictedBlock, after)
	if err != nil {
		return 0, err
	}
	if isBetterBlock {
		// If the predicted block is better, return its block number
		return predictedBlock.number, nil
	}

	// Calculate the difference between the current date and the predicted block's timestamp
	difference := date - predictedBlock.timestamp
	if blockTime == 0 {
		// If the block time is 0, set it to 1 to avoid division by zero
		blockTime = 1
	}

	// Calculate the number of blocks to skip based on the difference and the block time
	skip := int64(math.Ceil(float64(difference) / blockTime))
	if skip == 0 {
		// If skip is 0, set it to 1 or -1 based on the sign of the difference
		if difference < 0 {
			skip = -1
		} else {
			skip = 1
		}
	}

	// Get the next predicted block
	nextPredictedBlock, err := gb.getBlockWrapper(
		ctx,
		gb.getNextBlock(date, predictedBlock.number, skip),
	)
	if err != nil {
		return 0, err
	}

	// Calculate the new block time based on the difference between the predicted blocks
	blockTime = float64(math.Abs(
		(float64(predictedBlock.timestamp) - float64(nextPredictedBlock.timestamp)) /
			(float64(predictedBlock.number) - float64(nextPredictedBlock.number))))

	// Recursively call findBetterAdv with the updated block time
	return gb.findBetterAdv(ctx, date, nextPredictedBlock, after, blockTime)
}

//findBetter with moore parameters
func (gb *GoBlock) findBetterAdv(
	ctx context.Context,
	date int64,
	predictedBlock Block,
	after bool,
	blockTime float64,
) (int64, error) {
	isBetterBlock, err := gb.isBetterBlock(ctx, date, predictedBlock, after)
	if err != nil {
		return 0, err
	}
	if isBetterBlock {
		return predictedBlock.number, nil
	}
	difference := date - predictedBlock.timestamp
	if blockTime == 0 {
		blockTime = 1
	}
	skip := int64(math.Ceil(float64(difference) / blockTime))
	if skip == 0 {
		if difference < 0 {
			skip = -1
		} else {
			skip = 1
		}
	}
	nextPredictedBlock, err := gb.getBlockWrapper(
		ctx,
		gb.getNextBlock(date, predictedBlock.number, skip),
	)
	if err != nil {
		return 0, err
	}
	blockTime = float64(math.Abs(
		(float64(predictedBlock.timestamp) - float64(nextPredictedBlock.timestamp)) /
			(float64(predictedBlock.number) - float64(nextPredictedBlock.number))))
	return gb.findBetterAdv(ctx, date, nextPredictedBlock, after, blockTime)
}

func (gb *GoBlock) isBetterBlock(
	ctx context.Context,
	date int64,
	predictedBlock Block,
	after bool,
) (bool, error) {
	// blockTime is the timestamp of the predicted block
	blockTime := predictedBlock.timestamp

	if after {
		// If "after" is true, check if the predicted block is after the current date
		if blockTime < date {
			return false, nil
		}

		// Get the previous block
		previousBlock, err := gb.getBlockWrapper(ctx, predictedBlock.number-1)
		if err != nil {
			return false, err
		}

		// Check if the predicted block is after the current date and the previous block is before the current date
		if blockTime >= date && previousBlock.timestamp < date {
			return true, nil
		} else {
			if blockTime >= date {
				return false, nil
			}

			// If the predicted block is before the current date, get the next block
			var nextBlock Block
			nextBlock, err = gb.getBlockWrapper(ctx, predictedBlock.number+1)
			if err != nil {
				return false, err
			}

			// Check if the predicted block is before the current date and the next block is after the current date
			if blockTime < date && nextBlock.timestamp >= date {
				return true, nil
			}
		}
	}
	return false, nil
}


func (gb *GoBlock) getNextBlock(date int64, currentBlock int64, skip int64) int64 {
	var nextBlock int64
	nextBlock = currentBlock + skip
	if nextBlock > gb.latestBlock.number {
		nextBlock = gb.latestBlock.number
	}
	if slices.Contains(gb.checkedBlocks[date], nextBlock) {
		return gb.getNextBlock(date, currentBlock, skip)
	}
	gb.checkedBlocks[date] = append(gb.checkedBlocks[date], nextBlock)
	if nextBlock < 1 {
		return 1
	} else {
		return int64(nextBlock)
	}
}

func (gb *GoBlock) getBoundaries(ctx context.Context) error {
	var err error
	gb.latestBlock, err = gb.getBlockWrapper(ctx, -1)
	if err != nil {
		return err
	}
	gb.firstBlock, err = gb.getBlockWrapper(ctx, 1)
	if err != nil {
		return err
	}
	gb.blockTime = float64(
		gb.latestBlock.timestamp-gb.firstBlock.timestamp,
	) / float64(
		gb.latestBlock.number-1,
	)
	return nil
}

func (gb *GoBlock) getBlockWrapper(ctx context.Context, blockNumber int64) (Block, error) {
	if blockNumber == -1 {
		head, err := gb.getHeaderBlock(ctx)
		if err != nil {
			return Block{}, err
		}
		block, err := gb.getBlock(ctx, head)
		if err != nil {
			return Block{}, err
		}
		gb.savedBlocks[block.number] = block
		return gb.savedBlocks[block.number], nil
	}
	if gb.savedBlocks[blockNumber] != (Block{}) {
		return gb.savedBlocks[blockNumber], nil
	}
	block, err := gb.getBlock(ctx, blockNumber)
	if err != nil {
		return Block{}, err
	}
	gb.savedBlocks[blockNumber] = block
	return gb.savedBlocks[blockNumber], nil
}

func (gb *GoBlock) getBlock(ctx context.Context, blockNumber int64) (Block, error) {
	var block Block
	b, err := gb.client.BlockByNumber(ctx, big.NewInt(blockNumber))
	if err != nil {
		return Block{}, err
	}
	block = Block{number: int64(b.Number().Uint64()), timestamp: int64(b.Time())}
	return block, nil
}

func (gb *GoBlock) getHeaderBlock(ctx context.Context) (int64, error) {
	var header int64
	h, err := gb.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return 0, err
	}
	header = h.Number.Int64()
	return header, nil
}

// ex: _date -> "YYYY-MM-DDThh:mm:ss+hh:mm"
func (gb *GoBlock) DateToBlock(ctx context.Context, date string) (int64, error) {
	dateTime, err := time.Parse(time.RFC3339, date)
	if err != nil {
		return 0, err
	}
	timestamp := dateTime.Unix()
	block, err := gb.GetDate(ctx, timestamp)
	if err != nil {
		return 0, err
	}
	return block, nil
}

func (gb *GoBlock) BlockToDate(ctx context.Context, number int64) (string, error) {
	b, err := gb.client.BlockByNumber(ctx, big.NewInt(number))
	if err != nil {
		return "", err
	}
	t := int64(b.Time())
	tm := time.Unix(t, 0).Format(time.RFC3339)
	return tm, nil
}
