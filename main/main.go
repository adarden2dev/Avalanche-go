// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"fmt"
	"path"

	"github.com/ava-labs/gecko/nat"
	"github.com/ava-labs/gecko/node"
	"github.com/ava-labs/gecko/utils/crypto"
	"github.com/ava-labs/gecko/utils/logging"
)

// main is the primary entry point to Avalanche.
func main() {
	// Err is set based on the CLI arguments
	if Err != nil {
		fmt.Printf("parsing parameters returned with error %s\n", Err)
		return
	}

	config := Config.LoggingConfig
	config.Directory = path.Join(config.Directory, "node")
	factory := logging.NewFactory(config)
	defer factory.Close()

	log, err := factory.Make()
	if err != nil {
		fmt.Printf("starting logger failed with: %s\n", err)
		return
	}
	fmt.Println(gecko)

	defer func() { recover() }()

	defer log.Stop()
	defer log.StopOnPanic()
	defer Config.DB.Close()

	// Track if sybil control is enforced
	if !Config.EnableStaking && Config.EnableP2PTLS {
		log.Warn("Staking is disabled. Sybil control is not enforced.")
	}
	if !Config.EnableStaking && !Config.EnableP2PTLS {
		log.Warn("Staking and p2p encryption are disabled. Packet spoofing is possible.")
	}

	// Check if transaction signatures should be checked
	if !Config.EnableCrypto {
		log.Warn("transaction signatures are not being checked")
	}
	crypto.EnableCrypto = Config.EnableCrypto

	if err := Config.ConsensusParams.Valid(); err != nil {
		log.Fatal("consensus parameters are invalid: %s", err)
		return
	}

	// Track if assertions should be executed
	if Config.LoggingConfig.Assertions {
		log.Debug("assertions are enabled. This may slow down execution")
	}

	mapper := nat.NewPortMapper(log, Config.Nat)
	defer mapper.UnmapAllPorts()

	// Open staking port
	port, err := mapper.Map("TCP", Config.StakingLocalPort, "gecko-staking")
	if !Config.StakingIP.IsPrivate() {
		if err == nil {
			// The port was mapped and the ip is on a public network, the node
			// should be able to be connected to peers on this public network.
			Config.StakingIP.Port = port
		} else {
			// The port mapping errored, however it is possible the node is
			// connected directly to a public network.
			log.Warn("NAT traversal has failed. Unless the node is connected directly to a public network, the node will be able to connect to less nodes.")
		}
	} else {
		// The reported IP is private, so this node will not be discoverable.
		log.Warn("NAT traversal has failed. The node will be able to connect to less nodes.")
	}

	// Open the HTTP port iff the HTTP server is not listening on localhost
	if Config.HTTPHost != "127.0.0.1" && Config.HTTPHost != "localhost" {
		_, _ = mapper.Map("TCP", Config.HTTPPort, "gecko-http")
	}

	node := node.Node{}

	log.Debug("initializing node state")
	if err := node.Initialize(&Config, log, factory); err != nil {
		log.Fatal("error initializing node state: %s", err)
		return
	}

	defer node.Shutdown()

	log.Debug("dispatching node handlers")
	err = node.Dispatch()
	log.Debug("node dispatching returned with %s", err)
}
