package chain

import (
	"sync"

	"gitlab.com/thorchain/tss/go-tss/config"
)

// with each networks wait to wg done then process
var wg *sync.WaitGroup

type ChainConfig struct {
	validator string
	url       string
}

func NewChainConfig(url, validator string) *ChainConfig {
	return &ChainConfig{url: url, validator: validator}
}

type MetaData struct {
	IncTokenId string `json:"incTokenId"`
}

type ShieldTxData struct {
	RequestTx        string
	Amount           int64
	ToAddr           string
	IncognitoAddress string
	MetaData         []byte
}

type UnshieldTxData struct {
	RequestTx string // burn transaction from incognito chain. Tss can retrieve burn proof
	Amount    string
	ToAddr    string
}

type BaseChain struct {
	currentHeight    uint64
	ShieldTxsQueue   chan ShieldTxData
	UnshieldTxsQueue chan UnshieldTxData
	cfg              ChainConfig
	stopCh           chan struct{}
}

func NewBaseChain(cfg ChainConfig) *BaseChain {
	return &BaseChain{
		ShieldTxsQueue:   make(chan ShieldTxData),
		UnshieldTxsQueue: make(chan UnshieldTxData),
		cfg:              cfg,
		stopCh:           make(chan struct{}),
	}
}

func InitChains() map[string]Chain {

	wg = &sync.WaitGroup{}
	// Load from database here if bridge network has start then no need to add to wait group here
	// wg.Add(1)

	res := make(map[string]Chain) // 0 -> Incognitochain, 1 -> BTC, 2 -> ETH
	c := config.GetConfig()
	for k, v := range c.ChainsConfig {
		switch k {
		case BtcChainId:
			cfg := NewChainConfig(v.Url, "")
			res[k] = NewBtc(*NewBtcCfg(*cfg), *NewBaseChain(*cfg))
		case IncognitoChainId:
			cfg := NewChainConfig(v.Url, "")
			res[k] = NewIncognito(*NewIncognitoCfg(*cfg), *NewBaseChain(*cfg))
		case BridgeChainId:

		default:
			panic("cannot find chain id ")
		}
	}
	return res
}
