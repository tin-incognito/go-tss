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

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"

	"github.com/cosmos/cosmos-sdk/crypto"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	tmcrypto "github.com/tendermint/tendermint/crypto"
	tmsecp256k1 "github.com/tendermint/tendermint/crypto/secp256k1"
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

// Keys manages all the keys used by bridge network
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
	RelayerAddress string
}

func NewBridgeClientConfig(cfg *ChainClientConfig, relayerAddress string) *BridgeClientConfig {
	return &BridgeClientConfig{
		ChainClientConfig: *cfg,
		RelayerAddress:    relayerAddress,
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

func (b *BridgeClient) sendKeygenTx(creator, poolPk string, blame *types.Blame, input []string, keygenType int32, chains []string, height, keygenTime int64) error {
	keygenMsg, err := b.getKeygenStdTx(creator, poolPk, blame, input, keygenType, chains, height, keygenTime)
	if err != nil {
		return fmt.Errorf("fail to get keygen id: %w", err)
	}
	txId, err := b.broadcast(keygenMsg)
	if err != nil {
		return fmt.Errorf("fail to send the tx to bridge network: %w", err)
	}
	fmt.Println("bridge network tx hash", txId, "sign and send to bridge network successfully")
	return nil
}

func (b *BridgeClient) sendRegisterKeygenTx(creator, msg, signature string) error {
	registerKeygenMsg, err := b.getRegisterKeygenStdTx(creator, msg, signature)
	if err != nil {
		return fmt.Errorf("fail to get registerKeygenMsg id: %w", err)
	}
	txId, err := b.broadcast(registerKeygenMsg)
	if err != nil {
		return fmt.Errorf("fail to send the tx to bridge network: %w", err)
	}
	fmt.Println("bridge network tx hash", txId, "sign and send to bridge network successfully")
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

	builder.SetMsgs(msgs...)

	//builder.SetGasLimit(4000000000)
	builder.SetGasLimit(200000)
	err = clienttx.Sign(factory, ctx.GetFromName(), builder, true)
	fmt.Println("err:", err)
	if err != nil {
		return txId, err
	}

	txBytes, err := ctx.TxConfig.TxEncoder()(builder.GetTx())
	if err != nil {
		return txId, err
	}
	temp, err := ctx.TxConfig.TxJSONEncoder()(builder.GetTx())
	if err != nil {
		return txId, err
	}
	fmt.Println("temp:", string(temp))

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
	accountInfo, err := http.GetAccountInfo(b.cfg.Stateurl, accountAddress.String())
	if err != nil {
		return 0, 0, err
	}

	return uint64(accountInfo.AccountNumber), uint64(accountInfo.Sequence), nil
}

func (b *BridgeClient) getRegisterKeygenStdTx(creator, msg, signature string) (sdk.Msg, error) {
	return types.NewMsgRegisterTssPool(creator, msg, signature)
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

	ctx = ctx.WithNodeURI(b.cfg.RpcUrl)
	client, err := rpchttp.New(b.cfg.RpcUrl, "/websocket")
	if err != nil {
		panic(err)
	}
	ctx = ctx.WithClient(client)
	return ctx
}

// GetKeyringKeybase return keyring and key info
func GetKeyringKeybase(chainHomeFolder, password string) (ckeys.Keyring, error) {
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
func getKeybase(home string, reader io.Reader) (ckeys.Keyring, error) {
	cliDir := home
	if len(home) == 0 {
		usr, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("fail to get current user,err:%w", err)
		}
		cliDir = filepath.Join(usr.HomeDir, bridgeNetworkCliFolderName)
	}

	encodingConfig := app.MakeEncodingConfig()
	return ckeys.New(sdk.KeyringServiceName(), ckeys.BackendOS, cliDir, reader, encodingConfig.Marshaler)
}

// GetPrivateKey return the private key
func (k *Keys) GetPrivateKey() (cryptotypes.PrivKey, error) {
	// return k.kb.ExportPrivateKeyObject(k.signerName)
	privKeyArmor, err := k.kb.ExportPrivKeyArmor(k.signerName, k.password)
	if err != nil {
		return nil, err
	}
	priKey, _, err := crypto.UnarmorDecryptPrivKey(privKeyArmor, k.password)
	if err != nil {
		return nil, fmt.Errorf("fail to unarmor private key: %w", err)
	}
	return priKey, nil
}

// CosmosPrivateKeyToTMPrivateKey convert cosmos implementation of private key to tendermint private key
func CosmosPrivateKeyToTMPrivateKey(privateKey cryptotypes.PrivKey) tmcrypto.PrivKey {
	switch k := privateKey.(type) {
	case *secp256k1.PrivKey:
		return tmsecp256k1.PrivKey(k.Bytes())
	default:
		return nil
	}
}
