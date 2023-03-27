package config

import (
	"flag"
	"fmt"
	"strings"
	"time"

	maddr "github.com/multiformats/go-multiaddr"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type TssConfig struct {
	// Party Timeout defines how long do we wait for the party to form
	PartyTimeout time.Duration `mapstructure:"party_time_out"`
	// KeyGenTimeoutSeconds defines how long do we wait the keygen parties to pass messages along
	KeyGenTimeout time.Duration `mapstructure:"key_gen_timeout"`
	// KeySignTimeoutSeconds defines how long do we wait keysign
	KeySignTimeout time.Duration `mapstructure:"key_sign_timeout"`
	// Pre-parameter define the pre-parameter generations timeout
	PreParamTimeout time.Duration `mapstructure:"pre_param_timeout"`
	// enable the tss monitor
	EnableMonitor bool `mapstructure:"enable_monitor"`
}

// A new type we need for writing a custom flag parser
type addrList []maddr.Multiaddr

// String implement fmt.Stringer
func (al *addrList) String() string {
	addresses := make([]string, len(*al))
	for i, addr := range *al {
		addresses[i] = addr.String()
	}
	return strings.Join(addresses, ",")
}

// Set add the given value to addList
func (al *addrList) Set(value string) error {
	addr, err := maddr.NewMultiaddr(value)
	if err != nil {
		return err
	}
	*al = append(*al, addr)
	return nil
}

// Config is configuration for P2P
type P2pConfig struct {
	RendezvousString string   `mapstructure:"rendezvous_string"`
	Port             int      `mapstructure:"port"`
	BootstrapPeers   addrList `mapstructure:"bootstrap_peers"`
	ExternalIP       string   `mapstructure:"external_ip" `
}

type BridgeConfig struct {
	BlockUrl       string `mapstructure:"block_url"`
	StateUrl       string `mapstructure:"state_url"`
	RpcUrl         string `mapstructure:"rpc_url"`
	RelayerAddress string `mapstructure:"relayer_address"` // TODO: This is the bad way, try to improve later
	SignerName     string
	SignerPasswd   string // TODO: This is the bad way, try to improve later
}

type ChainConfig struct {
	Url string `mapstructure:"url"`
}

type Config struct {
	TssAddr    string `mapstructure:"tss_addr"`
	Help       bool   `mapstructure:"help"`
	LogLevel   string `mapstructure:"log_level"`
	PrettyLog  bool   `mapstructure:"pretty_log"`
	BaseFolder string `mapstructure:"base_folder"`

	BridgeConfig *BridgeConfig           `mapstructure:"bridge_config"`
	ChainsConfig map[string]*ChainConfig `mapstructure:"chains_config"`

	TssConfig *TssConfig `mapstructure:"tss_config"`
	P2pConfig *P2pConfig `mapstructure:"p2p_config"`
}

var c *Config

func GetConfig() *Config {
	return c
}

func InitConfig() error {
	viper.SetConfigName("local")     // name of config file (without extension)
	viper.SetConfigType("yaml")      // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath("./config/") // path to look for the config file in
	viper.AddConfigPath(".")         // path to look for the config file in
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			return fmt.Errorf("Config file was not found")
		} else {
			return err
			// Config file was found but another error was produced
		}
	} else {
		err = viper.Unmarshal(&c)
		if err != nil {
			return err
		}
	}

	c.readFromFlag()

	fmt.Println("c.LogLevel:", c.LogLevel)

	return nil
}

func (c *Config) readFromFlag() {
	// we setup the configure for the general configuration
	flag.StringVar(&c.TssAddr, "tss_addr", c.TssAddr, "tss address")
	flag.BoolVar(&c.Help, "h", false, "Display Help")
	flag.StringVar(&c.LogLevel, "log_level", c.LogLevel, "Log Level")
	flag.BoolVar(&c.PrettyLog, "pretty_log", false, "Enables unstructured prettified logging. This is useful for local debugging")
	flag.StringVar(&c.BaseFolder, "base_folder", c.BaseFolder, "home folder to store the keygen state file")

	// we setup the Tss parameter configuration
	flag.DurationVar(&c.TssConfig.KeyGenTimeout, "gentimeout", 30*time.Second, "keygen timeout")
	flag.DurationVar(&c.TssConfig.KeySignTimeout, "signtimeout", 30*time.Second, "keysign timeout")
	flag.DurationVar(&c.TssConfig.PreParamTimeout, "preparamtimeout", 5*time.Minute, "pre-parameter generation timeout")
	flag.BoolVar(&c.TssConfig.EnableMonitor, "enablemonitor", true, "enable the tss monitor")

	// we setup the p2p network configuration
	flag.StringVar(&c.P2pConfig.RendezvousString, c.P2pConfig.RendezvousString, "Asgard",
		"Unique string to identify group of nodes. Share this with your friends to let them connect with you")
	flag.IntVar(&c.P2pConfig.Port, "p2p_port", c.P2pConfig.Port, "listening port local")
	flag.StringVar(&c.P2pConfig.ExternalIP, "external-ip", c.P2pConfig.ExternalIP, "external IP of this node")
	flag.Var(&c.P2pConfig.BootstrapPeers, "peer", "Adds a peer multiaddress to the bootstrap list")
	flag.StringVar(&c.BridgeConfig.BlockUrl, "bridge_block_url", c.BridgeConfig.BlockUrl, "url for bridge chain")
	flag.StringVar(&c.BridgeConfig.StateUrl, "bridge_state_url", c.BridgeConfig.StateUrl, "url for bridge chain")
	flag.StringVar(&c.BridgeConfig.SignerName, "bridge_signer_name", c.BridgeConfig.SignerName, "signer name (validator name)")
	flag.StringVar(&c.BridgeConfig.SignerPasswd, "bridge_signer_password", c.BridgeConfig.SignerPasswd, "signer password")
	flag.StringVar(&c.BridgeConfig.RelayerAddress, "bridge_relayer_address", c.BridgeConfig.RelayerAddress, "relayer address")
	flag.StringVar(&c.BridgeConfig.RpcUrl, "bridge_rpc_url", c.BridgeConfig.RpcUrl, "url for rpc bridge chain")
	// using standard library "flag" package
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
}
