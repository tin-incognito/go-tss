package network

import (
	"bridge/app"
	"bridge/x/bridge/types"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
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

const (
	bridgeNetworkCliFolderName = ".bridge"
)

// Keys manages all the keys used by thorchain
type Keys struct {
	signerName string
	password   string // TODO this is a bad way , need to fix it
	kb         ckeys.Keyring
}

func NewKeys(signerName, password string, kb ckeys.Keyring) *Keys {
	return &Keys{
		signerName: signerName,
		password:   password,
		kb:         kb,
	}
}

func NewBridgeClient(blockUrl, stateUrl string, keys *Keys, cfg *BridgeClientConfig) *BridgeClient {
	return &BridgeClient{
		ChainClient:   *NewChainClient(&cfg.ChainClientConfig),
		cfg:           *cfg,
		broadcastLock: &sync.RWMutex{},
		blockHeight:   0,
		keys:          keys,
	}
}

type BridgeClientConfig struct {
	ChainClientConfig
}

func NewBridgeClientConfig(cfg *ChainClientConfig) *BridgeClientConfig {
	return &BridgeClientConfig{
		ChainClientConfig: *cfg,
	}
}

type BridgeClient struct {
	ChainClient
	cfg           BridgeClientConfig
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
	accountInfo, err := http.GetAccountInfo(b.cfg.BlockUrl, accountAddress.String())
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
	ctx = ctx.WithChainID(b.cfg.ChainId)
	ctx = ctx.WithHomeDir("")
	ctx = ctx.WithFromName(b.cfg.SignerName)
	accountAddress, _ := b.AccountAddress()
	ctx = ctx.WithFromAddress(accountAddress)
	ctx = ctx.WithBroadcastMode("sync")

	encodingConfig := app.MakeEncodingConfig()
	ctx = ctx.WithCodec(encodingConfig.Marshaler)
	ctx = ctx.WithInterfaceRegistry(encodingConfig.InterfaceRegistry)
	ctx = ctx.WithTxConfig(encodingConfig.TxConfig)
	ctx = ctx.WithLegacyAmino(encodingConfig.Amino)
	ctx = ctx.WithAccountRetriever(authtypes.AccountRetriever{})

	remote := b.cfg.Stateurl

	ctx = ctx.WithNodeURI(remote)
	client, err := rpchttp.New(remote, "/websocket")
	if err != nil {
		panic(err)
	}
	ctx = ctx.WithClient(client)
	return ctx
}

// GetKeyringKeybase return keyring and key info
func GetKeyringKeybase(chainHomeFolder, signerName, password string) (ckeys.Keyring, error) {
	if len(signerName) == 0 {
		return nil, fmt.Errorf("signer name is empty")
	}
	if len(password) == 0 {
		return nil, fmt.Errorf("password is empty")
	}

	buf := bytes.NewBufferString(password)
	// the library used by keyring is using ReadLine , which expect a new line
	buf.WriteByte('\n')
	kb, err := getKeybase(chainHomeFolder, buf)
	if err != nil {
		return nil, fmt.Errorf("fail to get keybase,err:%w", err)
	}
	// the keyring library which used by cosmos sdk , will use interactive terminal if it detect it has one
	// this will temporary trick it think there is no interactive terminal, thus will read the password from the buffer provided
	oldStdIn := os.Stdin
	defer func() {
		os.Stdin = oldStdIn
	}()
	os.Stdin = nil
	return kb, nil
}

// getKeybase will create an instance of Keybase
func getKeybase(thorchainHome string, reader io.Reader) (ckeys.Keyring, error) {
	cliDir := thorchainHome
	if len(thorchainHome) == 0 {
		usr, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("fail to get current user,err:%w", err)
		}
		cliDir = filepath.Join(usr.HomeDir, bridgeNetworkCliFolderName)
	}

	encodingConfig := app.MakeEncodingConfig()
	return ckeys.New(sdk.KeyringServiceName(), ckeys.BackendFile, cliDir, reader, encodingConfig.Marshaler)
}
