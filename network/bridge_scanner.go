package network

import (
	"bridge/x/bridge/types"
	"fmt"
	"time"

	"gitlab.com/thorchain/tss/go-tss/network/http"
)

type BridgeScanner struct {
	HasRegistered bool
	BlockScanner
	KeygenCh       chan *types.KeygenBlock
	RegisterKeygen chan *types.RegisterKeygen
	txOutCh        chan struct{}
}

func NewBridgeScanner(blockUrl, stateUrl string) *BridgeScanner {
	return &BridgeScanner{
		BlockScanner:   *NewBlockScanner(blockUrl, stateUrl),
		KeygenCh:       make(chan *types.KeygenBlock),
		RegisterKeygen: make(chan *types.RegisterKeygen),
		txOutCh:        make(chan struct{}),
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
			if time.Since(lastCheck) < BridgeNetworkBlockTime {
				continue
			}
			lastCheck = time.Now()
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
			} else {
				panic(err)
				if err == http.ErrConnectionRefused {
					time.Sleep(BridgeNetworkBlockTime)
					continue
				} else {
					if err != http.ErrNotFoundKeyGenBlock {
						fmt.Println("err:", err)
					}
				}
			}
			if !b.HasRegistered {
				registerKeygen, err := http.GetRegisterKeygen(b.stateUrl)
				if err == nil {
					b.HasRegistered = true
					fmt.Println("Get registerKeygen", b.stateUrl, nextBlock, "Detect registerKeygen")
					b.RegisterKeygen <- registerKeygen
				} else {
					panic(err)
					if err == http.ErrConnectionRefused {
						time.Sleep(BridgeNetworkBlockTime)
						continue
					} else {
						if err != http.ErrNotFoundRegisterKeyGen {
							fmt.Println("err:", err)
						}
					}
				}
			}

			b.currentBlock = nextBlock
			time.Sleep(BridgeNetworkBlockTime)
		}
	}
}
