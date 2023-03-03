package network

import (
	"bridge/app"
	"bridge/x/bridge/types"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"gitlab.com/thorchain/tss/go-tss/network/http"

	"github.com/cosmos/cosmos-sdk/client"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/spf13/pflag"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"

	ckeys "github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stypes "github.com/cosmos/cosmos-sdk/types"
)

// Keys manages all the keys used by thorchain
type Keys struct {
	signerName string
	password   string // TODO this is a bad way , need to fix it
	kb         ckeys.Keyring
}

func NewBridgeClient() *BridgeClient {
	return &BridgeClient{}
}

type BridgeClientConfig struct {
	ChainClientConfig
}

type BridgeClient struct {
	ChainClient
	stateUrl      string
	config        BridgeClientConfig
	broadcastLock *sync.RWMutex
	blockHeight   int64
	accountNumber uint64
	seqNumber     uint64

	keys *Keys

	lastBlockHeightCheck time.Time
	currentBlockHeight   int64
}

func (b *BridgeClient) sendKeygenTx(poolPk string, blame *types.Blame, input []string, keygenType int32, chains []string, height, keygenTime int64) error {

	var creator string
	keygenMsg, err := b.getKeygenStdTx(creator, poolPk, blame, input, keygenType, chains, height, keygenTime)
	if err != nil {
		return fmt.Errorf("fail to get keygen id: %w", err)
	}
	txId, err := b.broadcast(keygenMsg)
	if err != nil {
		return fmt.Errorf("fail to send the tx to thorchain: %w", err)
	}
	fmt.Println("thorchain hash", txId, "sign and send to thorchain successfully")
	return nil
}

func (b *BridgeClient) broadcast(msgs ...stypes.Msg) (string, error) {
	b.broadcastLock.Lock()
	defer b.broadcastLock.Unlock()
	txId := ""

	blockHeight, err := b.GetCurrentHeight()
	if err != nil {
		return txId, err
	}
	if blockHeight > b.currentBlockHeight {
		accountNumber, seqNum, err := b.getAccountNumberAndSequenceNumber()
		if err != nil {
			return txId, fmt.Errorf("fail to get account number and sequence number from bridge network : %w", err)
		}
		if seqNum > b.seqNumber {
			b.seqNumber = seqNum
		}
		b.accountNumber = accountNumber
		b.currentBlockHeight = blockHeight
	}

	flags := pflag.NewFlagSet("bridge", 0)

	ctx := b.GetContext()
	factory := clienttx.NewFactoryCLI(ctx, flags)
	factory = factory.WithAccountNumber(b.accountNumber)
	factory = factory.WithSequence(b.seqNumber)
	factory = factory.WithSignMode(signing.SignMode_SIGN_MODE_DIRECT)

	builder, err := factory.BuildUnsignedTx(msgs...)
	if err != nil {
		return txId, err
	}
	builder.SetGasLimit(4000000000)
	err = clienttx.Sign(factory, ctx.GetFromName(), builder, true)
	if err != nil {
		return txId, err
	}

	txBytes, err := ctx.TxConfig.TxEncoder()(builder.GetTx())
	if err != nil {
		return txId, err
	}

	// broadcast to a Tendermint node
	commit, err := ctx.BroadcastTx(txBytes)
	if err != nil {
		return txId, fmt.Errorf("fail to broadcast tx: %w", err)
	}
	if commit.Code > 0 {
		if commit.Code == 32 {
			// bad sequence number, fetch new one
			_, seqNum, _ := b.getAccountNumberAndSequenceNumber()
			if seqNum > 0 {
				b.seqNumber = seqNum
			}
		}
		if commit.Code != 6 {
			return txId, fmt.Errorf("fail to broadcast to bridge network,code:%d, log:%s", commit.Code, commit.RawLog)
		}
	}
	txId = commit.TxHash
	// increment seqNum
	atomic.AddUint64(&b.seqNumber, 1)

	return txId, nil
}

func (b *BridgeClient) getAccountNumberAndSequenceNumber() (uint64, uint64, error) {
	accountAddress, err := b.AccountAddress()
	if err != nil {
		return 0, 0, err
	}
	accountInfo, err := http.GetAccountInfo(b.stateUrl, accountAddress.String())
	if err != nil {
		return 0, 0, err
	}

	return uint64(accountInfo.AccountNumber), uint64(accountInfo.Sequence), nil
}

func (b *BridgeClient) getKeygenStdTx(creator, poolPubKey string, blame *types.Blame, inputPks []string, keygenType int32, chains []string, height, keygenTime int64) (sdk.Msg, error) {
	return types.NewMsgTssPool(creator, inputPks, poolPubKey, keygenType, height, blame, chains, keygenTime)
}

func (b *BridgeClient) AccountAddress() (sdk.AccAddress, error) {
	t, err := b.keys.kb.Key(b.keys.signerName)
	if err != nil {
		return nil, err
	}
	a, err := t.GetPubKey()
	if err != nil {
		return nil, err
	}
	return sdk.AccAddress(a.Address()), nil
}

func (b *BridgeClient) GetContext() client.Context {
	ctx := client.Context{}
	ctx = ctx.WithKeyring(b.keys.kb)
	ctx = ctx.WithChainID(b.config.ChainId)
	ctx = ctx.WithHomeDir(b.config.ChainHomeFolder)
	ctx = ctx.WithFromName(b.config.SignerName)
	accountAddress, _ := b.AccountAddress()
	ctx = ctx.WithFromAddress(accountAddress)
	ctx = ctx.WithBroadcastMode("sync")

	encodingConfig := app.MakeEncodingConfig()
	ctx = ctx.WithCodec(encodingConfig.Marshaler)
	ctx = ctx.WithInterfaceRegistry(encodingConfig.InterfaceRegistry)
	ctx = ctx.WithTxConfig(encodingConfig.TxConfig)
	ctx = ctx.WithLegacyAmino(encodingConfig.Amino)
	ctx = ctx.WithAccountRetriever(authtypes.AccountRetriever{})

	remote := b.config.ChainRPC
	if !strings.HasSuffix(b.config.ChainHost, "http") {
		remote = fmt.Sprintf("tcp://%s", remote)
	}
	ctx = ctx.WithNodeURI(remote)
	client, err := rpchttp.New(remote, "/websocket")
	if err != nil {
		panic(err)
	}
	ctx = ctx.WithClient(client)
	return ctx
}
