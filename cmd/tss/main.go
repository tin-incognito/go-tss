package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	golog "github.com/ipfs/go-log"
	"gitlab.com/thorchain/binance-sdk/common/types"

	"gitlab.com/thorchain/tss/go-tss/common"
	"gitlab.com/thorchain/tss/go-tss/conversion"
	"gitlab.com/thorchain/tss/go-tss/network"
	"gitlab.com/thorchain/tss/go-tss/p2p"
	"gitlab.com/thorchain/tss/go-tss/tss"
)

var (
	help       bool
	logLevel   string
	pretty     bool
	baseFolder string
	tssAddr    string
)

func main() {

	// Parse the cli into configuration structs
	tssConf, p2pConf, bConf, _ := parseFlags()
	if help {
		flag.PrintDefaults()
		return
	}
	// Setup logging
	golog.SetAllLoggers(golog.LevelInfo)
	_ = golog.SetLogLevel("tss-lib", "INFO")
	common.InitLog(logLevel, pretty, "tss_service")

	// Setup Bech32 Prefixes
	conversion.SetupBech32Prefix()
	// this is only need for the binance library
	if os.Getenv("NET") == "testnet" || os.Getenv("NET") == "mocknet" {
		types.Network = types.TestNetwork
	}

	kb, err := network.GetKeyringKeybase("", bConf.SignerName, bConf.SignerPasswd)
	if err != nil {
		panic(err)
	}

	keys := network.NewKeys(bConf.SignerName, bConf.SignerPasswd, kb)

	// setup TSS signing
	priKey, err := keys.GetPrivateKey()
	if err != nil {
		panic(err)
	}
	tmPriKey := network.CosmosPrivateKeyToTMPrivateKey(priKey)

	/*myValidator, err := kb.Key("validator3")*/
	/*if err != nil {*/
	/*panic(err)*/
	/*}*/
	/*pubKey, err := myValidator.GetPubKey()*/
	/*if err != nil {*/
	/*panic(err)*/
	/*}*/
	/*bech32PubKey, _ := legacybech32.MarshalPubKey(legacybech32.AccPK, pubKey)*/
	/*pk, _ := NewPubKey(bech32PubKey)*/
	/*fmt.Println("pk:", pk)*/

	// init tss module
	tss, err := tss.NewTss(
		p2p.AddrList(p2pConf.BootstrapPeers),
		p2pConf.Port,
		tmPriKey,
		p2pConf.RendezvousString,
		baseFolder,
		tssConf,
		nil,
		p2pConf.ExternalIP,
	)
	if nil != err {
		log.Fatal(err)
	}
	s := NewTssHttpServer(tssAddr, tss)
	go func() {
		if err := s.Start(); err != nil {
			fmt.Println(err)
		}
	}()

	signer, err := network.NewSigner(
		tss,
		bConf.BlockUrl, bConf.StateUrl,
		keys,
		network.NewBridgeClientConfig(network.NewChainClientConfig(
			network.BridgeChainId,
			bConf.BlockUrl, bConf.StateUrl, bConf.RpcUrl,
			bConf.SignerName, bConf.SignerPasswd,
		), bConf.RelayerAddress),
	)
	if err != nil {
		log.Fatal(err)
	}
	if err := signer.Start(); err != nil {
		log.Fatal(err)
	}
	chains := network.InitChains()
	observer, err := network.NewObserver(chains)
	if err != nil {
		log.Fatal(err)
	}
	// observers -> observer eth -> listen eth network
	// observers -> observer btc -> listen btc network
	if err := observer.Start(); err != nil {
		log.Fatal(err)
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("stop ")
	fmt.Println(s.Stop())
}

// parseFlags - Parses the cli flags
func parseFlags() (tssConf common.TssConfig, p2pConf p2p.Config, bConf common.BridgeConfig, chainConfs map[string]common.ChainConfig) {
	// we setup the configure for the general configuration
	flag.StringVar(&tssAddr, "tss-port", "127.0.0.1:8080", "tss port")
	flag.BoolVar(&help, "h", false, "Display Help")
	flag.StringVar(&logLevel, "loglevel", "info", "Log Level")
	flag.BoolVar(&pretty, "pretty-log", false, "Enables unstructured prettified logging. This is useful for local debugging")
	flag.StringVar(&baseFolder, "home", "", "home folder to store the keygen state file")

	// we setup the Tss parameter configuration
	flag.DurationVar(&tssConf.KeyGenTimeout, "gentimeout", 30*time.Second, "keygen timeout")
	flag.DurationVar(&tssConf.KeySignTimeout, "signtimeout", 30*time.Second, "keysign timeout")
	flag.DurationVar(&tssConf.PreParamTimeout, "preparamtimeout", 5*time.Minute, "pre-parameter generation timeout")
	flag.BoolVar(&tssConf.EnableMonitor, "enablemonitor", true, "enable the tss monitor")

	// we setup the p2p network configuration
	flag.StringVar(&p2pConf.RendezvousString, "rendezvous", "Asgard",
		"Unique string to identify group of nodes. Share this with your friends to let them connect with you")
	flag.IntVar(&p2pConf.Port, "p2p-port", 6668, "listening port local")
	flag.StringVar(&p2pConf.ExternalIP, "external-ip", "", "external IP of this node")
	flag.Var(&p2pConf.BootstrapPeers, "peer", "Adds a peer multiaddress to the bootstrap list")
	flag.StringVar(&bConf.BlockUrl, "bridge_block_url", "http://localhost:26657", "url for bridge chain")
	flag.StringVar(&bConf.StateUrl, "bridge_state_url", "http://localhost:1317", "url for bridge chain")
	flag.StringVar(&bConf.SignerName, "bridge_signer_name", os.Getenv("SIGNER_NAME"), "signer name (validator name)")
	flag.StringVar(&bConf.SignerPasswd, "bridge_signer_password", os.Getenv("SIGNER_PASSWD"), "signer password")
	flag.StringVar(&bConf.RelayerAddress, "bridge_relayer_address", "bridge1t00hhfcwn8ja9cv64yzal9mdcjepyc53w9y0ms", "relayer address")
	flag.StringVar(&bConf.RpcUrl, "bridge_rpc_url", "tcp://127.0.0.1:26657", "url for bridge chain")
	//flag.StringVar(&bConf.SignerPrivateKey, "bridge_signer_private_key", "", "unarmor signer private key")
	flag.Parse()
	return
}

/*type PubKey string*/

/*// NewPubKey create a new instance of PubKey*/
/*// key is bech32 encoded string*/
/*func NewPubKey(key string) (PubKey, error) {*/
/*if len(key) == 0 {*/
/*return "", nil*/
/*}*/
/*_, err := legacybech32.UnmarshalPubKey(legacybech32.AccPK, key)*/
/*if err != nil {*/
/*return "", fmt.Errorf("%s is not bech32 encoded pub key,err : %w", key, err)*/
/*}*/
/*return PubKey(key), nil*/
/*}*/
