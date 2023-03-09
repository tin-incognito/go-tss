package network

import (
	"encoding/json"
	"fmt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/rs/zerolog/log"
	"gitlab.com/thorchain/tss/go-tss/network/btcprocessors"
)

const (
	BTCBlockConfirmations = 8
)

type ChainConfig struct {
	validator string
}

type MetaData struct {
	IncTokenId string `json:"incTokenId"`
}

type ShieldTxData struct {
	RequestTx string
	Amount    int64
	ToAddr    string
	Memo      string
	MetaData  []byte
}

type UnshieldTxData struct {
	RequestTx string // burn transaction from incognito chain. Tss can retrieve burn proof
	Amount    string
	ToAddr    string
}

type Chain struct {
	ShieldTxsQueue   chan ShieldTxData
	UnshieldTxsQueue chan UnshieldTxData
	cfg              ChainConfig
	stopCh           chan struct{}
}

func InitChains() map[int]*Chain {
	res := make(map[int]*Chain) // 0 -> Incognitochain, 1 -> BTC, 2 -> ETH
	return res
}

func (c *Chain) Start() error {
	go c.scan()
	go c.processTxIns()
	return nil
}

func (c *Chain) processTxIns() {
	for {
		select {
		case <-c.stopCh:
			return
		case Tx := <-c.ShieldTxsQueue: // external network
			fmt.Println(Tx)
		case Tx := <-c.UnshieldTxsQueue: // Incognitochain
			fmt.Println(Tx)
		}
	}
}

func (c *Chain) scan() error {
	//TODO: @thach scan external network here then return value to TxsQueue channel

	// call json rpc to external network full node (from external network)
	// catch event deposit
	//c.ShieldTxsQueue <- t

	// sync latest state beacon (from Incognitochain)
	//c.UnshieldTxsQueue <- t

	// receive external network tx
	//t := struct{}{}

	// call bitcoin full node to get latest block height
	//
	btcClient, err := btcprocessors.BuildBTCClient()
	if err != nil {
		log.Warn().Int("can not init btc client "+err.Error(), 1)
		return nil
	}

	// todo: update network configuration
	processBtcDeposits(btcClient, c, &chaincfg.TestNet3Params, c.cfg.validator)

	return nil
}

func processBtcDeposits(btcClient *rpcclient.Client, c *Chain, chainParams *chaincfg.Params, validatorAdd string) {
	// todo: query current scanned bitcoin height
	currentBtcHeight := int64(0)
	btcBestHeight, err := btcClient.GetBlockCount()
	if err != nil {
		log.Warn().Int("get bitcoin best height failed "+err.Error(), 2)
		return
	}
	if currentBtcHeight >= btcBestHeight-int64(BTCBlockConfirmations) {
		log.Warn().Int("get bitcoin best height failed "+err.Error(), 2)
		return
	}
	// process txs in the block
	for i := currentBtcHeight + 1; i <= btcBestHeight; i++ {
		blockHash, err := btcClient.GetBlockHash(i)
		if err != nil {
			log.Warn().Int("get bitcoin block hash by height failed "+err.Error(), 2)
			return
		}
		block, err := btcClient.GetBlock(blockHash)
		if err != nil {
			log.Warn().Int("get bitcoin block hash by height failed "+err.Error(), 2)
			return
		}
		for _, tx := range block.Transactions {
			for _, out := range tx.TxOut {
				addrStr, err := btcprocessors.ExtractPaymentAddrStrFromPkScript(out.PkScript, chainParams)
				if err != nil {
					log.Warn().Int("could not extract payment address string from pkscript with err "+err.Error(), 2)
					continue
				}
				if addrStr != validatorAdd {
					continue
				}

				/// todo: update btc id
				metaData, _ := json.Marshal(MetaData{
					IncTokenId: "",
				})

				memo, err := btcprocessors.ExtractAttachedMsgFromTx(tx)
				if err != nil {
					log.Warn().Int("could not extract memo from tx with err "+err.Error(), 2)
					continue
				}
				// todo: validate incognito payment address in memo

				c.ShieldTxsQueue <- ShieldTxData{
					RequestTx: tx.TxHash().String(),
					Amount:    out.Value,
					ToAddr:    addrStr,
					Memo:      memo,
					MetaData:  metaData,
				}
			}
		}
	}
}
