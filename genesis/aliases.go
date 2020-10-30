// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package genesis

import (
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/vms/avm"
	"github.com/ava-labs/avalanchego/vms/nftfx"
	"github.com/ava-labs/avalanchego/vms/platformvm"
	"github.com/ava-labs/avalanchego/vms/propertyfx"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
	"github.com/ava-labs/avalanchego/vms/timestampvm"
)

// Aliases returns the default aliases based on the network ID
func Aliases(genesisBytes []byte) (map[string][]string, map[[32]byte][]string, map[[32]byte][]string, error) {
	generalAliases := map[string][]string{
		"vm/" + platformvm.ID.String():             {"vm/platform"},
		"vm/" + avm.ID.String():                    {"vm/avm"},
		"vm/" + EVMID.String():                     {"vm/evm"},
		"vm/" + timestampvm.ID.String():            {"vm/timestamp"},
		"bc/" + constants.PlatformChainID.String(): {"P", "platform", "bc/P", "bc/platform"},
	}
	chainAliases := map[[32]byte][]string{
		constants.PlatformChainID: {"P", "platform"},
	}
	vmAliases := map[[32]byte][]string{
		platformvm.ID:  {"platform"},
		avm.ID:         {"avm"},
		EVMID:          {"evm"},
		timestampvm.ID: {"timestamp"},
		secp256k1fx.ID: {"secp256k1fx"},
		nftfx.ID:       {"nftfx"},
		propertyfx.ID:  {"propertyfx"},
	}

	genesis := &platformvm.Genesis{} // TODO let's not re-create genesis to do aliasing
	if err := platformvm.GenesisCodec.Unmarshal(genesisBytes, genesis); err != nil {
		return nil, nil, nil, err
	}
	if err := genesis.Initialize(); err != nil {
		return nil, nil, nil, err
	}

	for _, chain := range genesis.Chains {
		uChain := chain.UnsignedTx.(*platformvm.UnsignedCreateChainTx)
		switch {
		case avm.ID.Equals(uChain.VMID):
			generalAliases["bc/"+chain.ID().String()] = []string{"X", "avm", "bc/X", "bc/avm"}
			chainAliases[chain.ID()] = []string{"X", "avm"}
		case EVMID.Equals(uChain.VMID):
			generalAliases["bc/"+chain.ID().String()] = []string{"C", "evm", "bc/C", "bc/evm"}
			chainAliases[chain.ID()] = []string{"C", "evm"}
		case timestampvm.ID.Equals(uChain.VMID):
			generalAliases["bc/"+chain.ID().String()] = []string{"bc/timestamp"}
			chainAliases[chain.ID()] = []string{"timestamp"}
		}
	}
	return generalAliases, chainAliases, vmAliases, nil
}
