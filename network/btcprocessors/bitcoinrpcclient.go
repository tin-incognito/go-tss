package btcprocessors

import (
	"bytes"
	"fmt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"os"

	"github.com/btcsuite/btcd/rpcclient"
)

func BuildBTCClient() (*rpcclient.Client, error) {
	connCfg := &rpcclient.ConnConfig{
		Host:         os.Getenv("BTC_NODE_HOST"),
		User:         os.Getenv("BTC_NODE_USERNAME"),
		Pass:         os.Getenv("BTC_NODE_PASSWORD"),
		HTTPPostMode: true,                                     // Bitcoin core only supports HTTP POST mode
		DisableTLS:   !(os.Getenv("BTC_NODE_HTTPS") == "true"), // Bitcoin core does not provide TLS by default
	}
	return rpcclient.New(connCfg, nil)
}

func ExtractPaymentAddrStrFromPkScript(pkScript []byte, params *chaincfg.Params) (string, error) {
	_, addrs, _, err := txscript.ExtractPkScriptAddrs(pkScript, params)
	if err != nil {
		return "", err
	}
	if len(addrs) == 0 {
		return "", nil
	}
	return addrs[0].EncodeAddress(), nil
}

func ExtractAttachedMsgFromTx(msgTx *wire.MsgTx) (string, error) {
	opReturnPrefix := []byte{
		txscript.OP_RETURN,
	}
	for _, txOut := range msgTx.TxOut {
		if txOut.Value != 0 || !bytes.HasPrefix(txOut.PkScript, opReturnPrefix) {
			continue
		}
		opReturnPkScript := txOut.PkScript
		if len(opReturnPkScript) < 5 {
			return "", fmt.Errorf("Memo is invalid")
		}
		first_byte := opReturnPkScript[1]
		if first_byte <= 75 {
			return string(opReturnPkScript[2:]), nil
		} else if first_byte == 76 { //0x4c
			return string(opReturnPkScript[3:]), nil
		} else if first_byte == 77 { //0x4d
			return string(opReturnPkScript[4:]), nil
		}
	}
	return "", nil
}
