package test

import (
	"bridge/app"
	"fmt"
	"regexp"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32/legacybech32"
)

func initSDKConfig() {
	// Set prefixes
	accountPubKeyPrefix := app.AccountAddressPrefix + "pub"
	validatorAddressPrefix := app.AccountAddressPrefix + "valoper"
	validatorPubKeyPrefix := app.AccountAddressPrefix + "valoperpub"
	consNodeAddressPrefix := app.AccountAddressPrefix + "valcons"
	consNodePubKeyPrefix := app.AccountAddressPrefix + "valconspub"

	// Set and seal config
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(app.AccountAddressPrefix, accountPubKeyPrefix)
	config.SetBech32PrefixForValidator(validatorAddressPrefix, validatorPubKeyPrefix)
	config.SetBech32PrefixForConsensusNode(consNodeAddressPrefix, consNodePubKeyPrefix)
	config.Seal()
}

func TestGenerateBTCAddressFromPubKey(t *testing.T) {
	initSDKConfig()
	poolPubKey := "bridgepub1addwnpepqd7kpulw26n9pwesvus80kgd3zaf6h98xg0kfwdmr4cdaeyl8jckjqjrxlw"
	pk, err := legacybech32.UnmarshalPubKey(legacybech32.AccPK, poolPubKey)
	if err != nil {
		panic(err)
	}
	var net *chaincfg.Params
	//net = &chaincfg.RegressionNetParams
	net = &chaincfg.TestNet3Params
	addr, err := btcutil.NewAddressWitnessPubKeyHash(pk.Address().Bytes(), net)
	if err != nil {
		panic(err)
	}
	temp, err := NewAddress(addr.String())
	if err != nil {
		panic(err)
	}
	fmt.Println("temp:", temp)
}

var alphaNumRegex = regexp.MustCompile("^[:A-Za-z0-9]*$")

func NewAddress(address string) (string, error) {
	if len(address) == 0 {
		return "", nil
	}
	if !alphaNumRegex.MatchString(address) {
		return "", fmt.Errorf("address format not supported: %s", address)
	}
	_, err := btcutil.DecodeAddress(address, &chaincfg.TestNet3Params)
	if err != nil {
		return "", err
	}
	return address, nil
}
