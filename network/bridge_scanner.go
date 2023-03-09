package network

import (
	"bridge/x/bridge/types"
	"fmt"
	"time"

	"gitlab.com/thorchain/tss/go-tss/network/http"
)

type BridgeScanner struct {
	BlockScanner
	KeygenCh chan *types.KeygenBlock
	txOutCh  chan struct{}
}

func NewBridgeScanner(blockUrl, stateUrl string) *BridgeScanner {
	return &BridgeScanner{
		BlockScanner: *NewBlockScanner(blockUrl, stateUrl),
		KeygenCh:     make(chan *types.KeygenBlock),
		txOutCh:      make(chan struct{}),
	}
}

func (b *BridgeScanner) Start() error {
	//go b.Scanner.Start()
	go b.scanKeygenBlock()
	return nil
}

func (b *BridgeScanner) scanKeygenBlock() error {
	fmt.Println("start scan keygen block")
	lastCheck := time.Now().Add(-BridgeNetworkBlockTime)
	for {
		select {
		case <-b.stopCh:

		default:
			nextBlock := b.currentBlock + 1
			if time.Since(lastCheck) >= BridgeNetworkBlockTime {
				lastCheck = time.Now()
				continue
			}
			chainCurrentHeight, err := b.BlockScanner.GetCurrentHeight()
			if err != nil {
				time.Sleep(BridgeNetworkBlockTime)
				continue
			}
			if chainCurrentHeight < nextBlock {
				time.Sleep(BridgeNetworkBlockTime)
				continue
			}
			keygenBlock, err := http.GetKeygenBlock(b.stateUrl, nextBlock)
			if err == nil {
				fmt.Println("Get keygenBlock", b.stateUrl, nextBlock, "Detect keygen block")
				b.KeygenCh <- keygenBlock
			}
			b.currentBlock = nextBlock
			time.Sleep(BridgeNetworkBlockTime)
		}
	}
}
