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
	lastMimirCheck := time.Now().Add(-BridgeNetworkBlockTime)
	for {
		select {
		case <-b.stopCh:

		default:
			nextBlock := b.BlockScanner.currentBlock + 1
			if time.Since(lastMimirCheck) >= BridgeNetworkBlockTime {
				lastMimirCheck = time.Now()
			}
			chainCurrentHeight, err := b.BlockScanner.GetCurrentHeight()
			if err != nil {
				return err
			}
			if chainCurrentHeight < nextBlock {
				time.Sleep(BridgeNetworkBlockTime)
				continue
			}
			keygenBlock, err := http.GetKeygenBlock(b.stateUrl, chainCurrentHeight)
			if err == nil {
				b.KeygenCh <- keygenBlock
			}
			b.currentBlock = nextBlock
		}
	}
}
