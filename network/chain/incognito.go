package chain

import (
	"fmt"
	"time"

	"gitlab.com/thorchain/tss/go-tss/network/http"
)

type Incognito struct {
	BaseChain
	cfg IncognitoCfg
}

type IncognitoCfg struct {
	ChainConfig
}

func NewIncognitoCfg(c ChainConfig) *IncognitoCfg {
	return &IncognitoCfg{ChainConfig: c}
}

func NewIncognito(cfg IncognitoCfg, baseChain BaseChain) *Incognito {
	return &Incognito{
		BaseChain: baseChain,
		cfg:       cfg,
	}
}

func (i *Incognito) Start() error {
	go i.Scan()
	go i.ProcessTxsIn()
	return nil
}

type RegisterKeygen struct {
	PoolPubKey string
	Members    []string
}

func (i *Incognito) Scan() error {
	fmt.Println("start scan incognito block")
	lastCheck := time.Now().Add(-IncognitoBlockTime)
	for {
		select {
		case <-i.stopCh:

		default:
			nextHeight := i.currentHeight + 1
			if time.Since(lastCheck) >= IncognitoBlockTime {
				lastCheck = time.Now()
				continue
			}
			chainCurrentHeight, err := http.GetCurrentBeaconHeight(i.cfg.url)
			if err != nil {
				time.Sleep(IncognitoBlockTime)
				continue
			}
			if chainCurrentHeight < nextHeight {
				time.Sleep(IncognitoBlockTime)
				continue
			}
			i.ParseAndProcessInstructions(nextHeight)
			i.currentHeight = nextHeight
			time.Sleep(IncognitoBlockTime)
		}
	}
}

func (i *Incognito) ProcessTxsIn() {
	/*for {*/
	/*select {*/
	/*case <-i.stopCh:*/
	/*return*/
	/*case Tx := <-i.ShieldTxsQueue: // external network*/
	/*fmt.Println(Tx)*/
	/*case Tx := <-i.UnshieldTxsQueue: // Incognitochain*/
	/*fmt.Println(Tx)*/
	/*}*/
	/*}*/
}

func (i *Incognito) ParseAndProcessInstructions(beaconHeight uint64) {
	instructions, err := http.GetBeaconInstructions(i.cfg.url, beaconHeight)
	if err == nil {
		for _, instruction := range instructions {
			switch instruction[0] {
			case "200":
				//TODO: add staking
			case "369":
				fmt.Println("receive start network instruction")
				//TODO: Start network
				// Check for valid bridge pubkey here
				// wg.Done() // Incognito has start network start producing block now
			case "201":
				//TODO: unshield instruction
			}
			//fmt.Println("Get keygenBlock", b.stateUrl, nextBlock, "Detect keygen block")
			//i.KeygenCh <- keygenBlock
		}

	} else {
		if err == http.ErrConnectionRefused {
			time.Sleep(IncognitoBlockTime)
			return
		} else {
			if err != http.ErrNotFoundKeyGenBlock {
				fmt.Println("err:", err)
			}
		}
	}
}
