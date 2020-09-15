// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package constants

import (
	"regexp"

	"github.com/ava-labs/avalanchego/ids"
)

// Const variables to be exported
const (
	ManhattenID uint32 = 0
	MainnetID   uint32 = 1
	CascadeID   uint32 = 2
	DenaliID    uint32 = 3
	EverestID   uint32 = 4

	TestnetID  uint32 = ManhattenID
	UnitTestID uint32 = 10
	LocalID    uint32 = 12345

	ManhattenName = "manhatten"
	MainnetName   = "mainnet"
	CascadeName   = "cascade"
	DenaliName    = "denali"
	EverestName   = "everest"
	TestnetName   = "testnet"
	UnitTestName  = "testing"
	LocalName     = "local"

	ManhattenHRP = "man"
	MainnetHRP   = "avax"
	CascadeHRP   = "cascade"
	DenaliHRP    = "denali"
	EverestHRP   = "everest"
	UnitTestHRP  = "testing"
	LocalHRP     = "local"
	FallbackHRP  = "custom"
)

// Variables to be exported
var (
	PrimaryNetworkID = ids.Empty
	PlatformChainID  = ids.Empty

	NetworkIDToNetworkName = map[uint32]string{
		ManhattenID: ManhattenName,
		MainnetID:   MainnetName,
		CascadeID:   CascadeName,
		DenaliID:    DenaliName,
		EverestID:   EverestName,
		UnitTestID:  UnitTestName,
		LocalID:     LocalName,
	}
	NetworkNameToNetworkID = map[string]uint32{
		ManhattenName: ManhattenID,
		MainnetName:   MainnetID,
		CascadeName:   CascadeID,
		DenaliName:    DenaliID,
		EverestName:   EverestID,
		TestnetName:   TestnetID,
		UnitTestName:  UnitTestID,
		LocalName:     LocalID,
	}

	NetworkIDToHRP = map[uint32]string{
		ManhattenID: ManhattenHRP,
		MainnetID:   MainnetHRP,
		CascadeID:   CascadeHRP,
		DenaliID:    DenaliHRP,
		EverestID:   EverestHRP,
		UnitTestID:  UnitTestHRP,
		LocalID:     LocalHRP,
	}
	NetworkHRPToNetworkID = map[string]uint32{
		ManhattenHRP: ManhattenID,
		MainnetHRP:   MainnetID,
		CascadeHRP:   CascadeID,
		DenaliHRP:    DenaliID,
		EverestHRP:   EverestID,
		UnitTestHRP:  UnitTestID,
		LocalHRP:     LocalID,
	}

	ValidNetworkName = regexp.MustCompile(`network-[0-9]+`)
)

// GetHRP returns the Human-Readable-Part of bech32 addresses for a networkID
func GetHRP(networkID uint32) string {
	if hrp, ok := NetworkIDToHRP[networkID]; ok {
		return hrp
	}
	return FallbackHRP
}
