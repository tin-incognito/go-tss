package network

import (
	"fmt"
	"time"

	"gitlab.com/thorchain/tss/go-tss/network/http"
)

type BlockScanner struct {
	blockUrl     string // 26657
	stateUrl     string // 1317
	stopCh       chan struct{}
	currentBlock int64
}

func NewBlockScanner(blockUrl, stateUrl string) *BlockScanner {
	return &BlockScanner{
		currentBlock: 0,
		stopCh:       make(chan struct{}),
		blockUrl:     blockUrl,
		stateUrl:     stateUrl,
	}
}

func (b *BlockScanner) GetCurrentHeight() (int64, error) {
	return http.GetCurrentHeight(b.blockUrl)
}

func (b *BlockScanner) Start() error {
	fmt.Println("Start block scanner")
	return nil
}

func (b *BlockScanner) scanBlocks() error {
	fmt.Println("Start scan blocks")
	lastMimirCheck := time.Now().Add(-BridgeNetworkBlockTime)
	for {
		select {
		case <-b.stopCh:

		default:
			nextBlock := b.currentBlock + 1
			if time.Since(lastMimirCheck) >= BridgeNetworkBlockTime {
				lastMimirCheck = time.Now()
			}
			chainCurrentHeight, err := b.GetCurrentHeight()
			if err != nil {
				return err
			}
			if chainCurrentHeight < nextBlock {
				time.Sleep(BridgeNetworkBlockTime)
				continue
			}
		}
	}
}
