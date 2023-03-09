package conversion

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func SetupBech32Prefix() {
	config := sdk.GetConfig()
	// thorchain will import go-tss as a library , thus this is not needed, we copy the prefix here to avoid go-tss to import thorchain
	config.SetBech32PrefixForAccount("bridge", "bridgepub")
	config.SetBech32PrefixForValidator("bridgev", "bridgevpub")
	config.SetBech32PrefixForConsensusNode("bridgec", "bridgecpub")
}
