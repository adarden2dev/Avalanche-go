// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chains

import (
	"fmt"
	"time"

	"github.com/ava-labs/gecko/api"
	"github.com/ava-labs/gecko/api/keystore"
	"github.com/ava-labs/gecko/chains/atomic"
	"github.com/ava-labs/gecko/database"
	"github.com/ava-labs/gecko/database/prefixdb"
	"github.com/ava-labs/gecko/ids"
	"github.com/ava-labs/gecko/network"
	"github.com/ava-labs/gecko/snow"
	"github.com/ava-labs/gecko/snow/consensus/snowball"
	"github.com/ava-labs/gecko/snow/engine/avalanche"
	"github.com/ava-labs/gecko/snow/engine/avalanche/state"
	"github.com/ava-labs/gecko/snow/engine/common"
	"github.com/ava-labs/gecko/snow/engine/common/queue"
	"github.com/ava-labs/gecko/snow/networking/router"
	"github.com/ava-labs/gecko/snow/networking/sender"
	"github.com/ava-labs/gecko/snow/networking/timeout"
	"github.com/ava-labs/gecko/snow/triggers"
	"github.com/ava-labs/gecko/snow/validators"
	"github.com/ava-labs/gecko/utils/logging"
	"github.com/ava-labs/gecko/vms"
	"github.com/ava-labs/gecko/vms/avm"

	avacon "github.com/ava-labs/gecko/snow/consensus/avalanche"
	avaeng "github.com/ava-labs/gecko/snow/engine/avalanche"

	smcon "github.com/ava-labs/gecko/snow/consensus/snowman"
	smeng "github.com/ava-labs/gecko/snow/engine/snowman"
)

const (
	defaultChannelSize = 1000
	gossipFrequency    = 10 * time.Second
	shutdownTimeout    = 1 * time.Second
)

// Manager manages the chains running on this node.
// It can:
//   * Create a chain
//   * Add a registrant. When a chain is created, each registrant calls
//     RegisterChain with the new chain as the argument.
//   * Get the aliases associated with a given chain.
//   * Get the ID of the chain associated with a given alias.
type Manager interface {
	// Return the router this Manager is using to route consensus messages to chains
	Router() router.Router

	// Create a chain in the future
	CreateChain(ChainParameters)

	// Create a chain now
	ForceCreateChain(ChainParameters)

	// Add a registrant [r]. Every time a chain is
	// created, [r].RegisterChain([new chain]) is called
	AddRegistrant(Registrant)

	// Given an alias, return the ID of the chain associated with that alias
	Lookup(string) (ids.ID, error)

	// Given an alias, return the ID of the VM associated with that alias
	LookupVM(string) (ids.ID, error)

	// Return the aliases associated with a chain
	Aliases(ids.ID) []string

	// Add an alias to a chain
	Alias(ids.ID, string) error

	// Returns true iff the chain with the given ID exists and is finished bootstrapping
	IsBootstrapped(ids.ID) bool

	Shutdown()
}

// ChainParameters defines the chain being created
type ChainParameters struct {
	ID          ids.ID   // The ID of the chain being created
	SubnetID    ids.ID   // ID of the subnet that validates this chain
	GenesisData []byte   // The genesis data of this chain's ledger
	VMAlias     string   // The ID of the vm this chain is running
	FxAliases   []string // The IDs of the feature extensions this chain is running

	CustomBeacons validators.Set // Should only be set if the default beacons can't be used.
}

type chain struct {
	Engine  common.Engine
	Handler *router.Handler
	Ctx     *snow.Context
	VM      interface{}
	Beacons validators.Set
}

type manager struct {
	// Note: The string representation of a chain's ID is also considered to be an alias of the chain
	// That is, [chainID].String() is an alias for the chain, too
	ids.Aliaser

	stakingEnabled  bool // True iff the network has staking enabled
	log             logging.Logger
	logFactory      logging.Factory
	vmManager       vms.Manager // Manage mappings from vm ID --> vm
	decisionEvents  *triggers.EventDispatcher
	consensusEvents *triggers.EventDispatcher
	db              database.Database
	chainRouter     router.Router      // Routes incoming messages to the appropriate chain
	net             network.Network    // Sends consensus messages to other validators
	timeoutManager  *timeout.Manager   // Manages request timeouts when sending messages to other validators
	consensusParams avacon.Parameters  // The consensus parameters (alpha, beta, etc.) for new chains
	validators      validators.Manager // Validators validating on this chain
	registrants     []Registrant       // Those notified when a chain is created
	nodeID          ids.ShortID        // The ID of this node
	networkID       uint32             // ID of the network this node is connected to
	server          *api.Server        // Handles HTTP API calls
	keystore        *keystore.Keystore
	sharedMemory    *atomic.SharedMemory

	// Key: Chain's ID
	// Value: The chain
	chains map[[32]byte]*router.Handler

	unblocked     bool
	blockedChains []ChainParameters
}

// New returns a new Manager where:
//     <db> is this node's database
//     <sender> sends messages to other validators
//     <validators> validate this chain
// TODO: Make this function take less arguments
func New(
	stakingEnabled bool,
	log logging.Logger,
	logFactory logging.Factory,
	vmManager vms.Manager,
	decisionEvents *triggers.EventDispatcher,
	consensusEvents *triggers.EventDispatcher,
	db database.Database,
	rtr router.Router,
	net network.Network,
	consensusParams avacon.Parameters,
	validators validators.Manager,
	nodeID ids.ShortID,
	networkID uint32,
	server *api.Server,
	keystore *keystore.Keystore,
	sharedMemory *atomic.SharedMemory,
) Manager {
	timeoutManager := timeout.Manager{}
	timeoutManager.Initialize(timeout.DefaultRequestTimeout)
	go log.RecoverAndPanic(timeoutManager.Dispatch)

	rtr.Initialize(log, &timeoutManager, gossipFrequency, shutdownTimeout)

	m := &manager{
		stakingEnabled:  stakingEnabled,
		log:             log,
		logFactory:      logFactory,
		vmManager:       vmManager,
		decisionEvents:  decisionEvents,
		consensusEvents: consensusEvents,
		db:              db,
		chainRouter:     rtr,
		net:             net,
		timeoutManager:  &timeoutManager,
		consensusParams: consensusParams,
		validators:      validators,
		nodeID:          nodeID,
		networkID:       networkID,
		server:          server,
		keystore:        keystore,
		sharedMemory:    sharedMemory,
		chains:          make(map[[32]byte]*router.Handler),
	}
	m.Initialize()
	return m
}

// Router that this chain manager is using to route consensus messages to chains
func (m *manager) Router() router.Router { return m.chainRouter }

// Create a chain
func (m *manager) CreateChain(chain ChainParameters) {
	if !m.unblocked {
		m.blockedChains = append(m.blockedChains, chain)
	} else {
		m.ForceCreateChain(chain)
	}
}

// Create a chain
func (m *manager) ForceCreateChain(chainParams ChainParameters) {
	m.log.Info("creating chain:\n"+
		"    ID: %s\n"+
		"    VMID:%s",
		chainParams.ID,
		chainParams.VMAlias,
	)

	// Assert that there isn't already a chain with an alias in [chain].Aliases
	// (Recall that the string repr. of a chain's ID is also an alias for a chain)
	if alias, isRepeat := m.isChainWithAlias(chainParams.ID.String()); isRepeat {
		m.log.Error("there is already a chain with alias '%s'. Chain not created.", alias)
		return
	}

	chain, err := m.buildChain(chainParams)
	if err != nil {
		m.log.Error("Error while creating new chain: %s", err)
		return
	}

	// Associate the newly created chain with its default alias
	m.log.AssertNoError(m.Alias(chainParams.ID, chainParams.ID.String()))

	// Notify those that registered to be notified when a new chain is created
	m.notifyRegistrants(chain.Ctx, chain.VM)
}

// Create a chain
func (m *manager) buildChain(chainParams ChainParameters) (*chain, error) {
	vmID, err := m.vmManager.Lookup(chainParams.VMAlias)
	if err != nil {
		return nil, fmt.Errorf("error while looking up VM: %s", err)
	}

	primaryAlias, err := m.PrimaryAlias(chainParams.ID)
	if err != nil {
		primaryAlias = chainParams.ID.String()
	}

	// Create the log and context of the chain
	chainLog, err := m.logFactory.MakeChain(primaryAlias, "")
	if err != nil {
		return nil, fmt.Errorf("error while creating chain's log %s", err)
	}

	ctx := &snow.Context{
		NetworkID:           m.networkID,
		ChainID:             chainParams.ID,
		Log:                 chainLog,
		DecisionDispatcher:  m.decisionEvents,
		ConsensusDispatcher: m.consensusEvents,
		NodeID:              m.nodeID,
		HTTP:                m.server,
		Keystore:            m.keystore.NewBlockchainKeyStore(chainParams.ID),
		SharedMemory:        m.sharedMemory.NewBlockchainSharedMemory(chainParams.ID),
		BCLookup:            m,
	}

	// Get a factory for the vm we want to use on our chain
	vmFactory, err := m.vmManager.GetVMFactory(vmID)
	if err != nil {
		return nil, fmt.Errorf("error while getting vmFactory: %s", err)
	}

	// Create the chain
	vm, err := vmFactory.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("error while creating vm: %s", err)
	}
	// TODO: Shutdown VM if an error occurs

	fxs := make([]*common.Fx, len(chainParams.FxAliases))
	for i, fxAlias := range chainParams.FxAliases {
		fxID, err := m.vmManager.Lookup(fxAlias)
		if err != nil {
			return nil, fmt.Errorf("error while looking up Fx: %s", err)
		}

		// Get a factory for the fx we want to use on our chain
		fxFactory, err := m.vmManager.GetVMFactory(fxID)
		if err != nil {
			return nil, fmt.Errorf("error while getting fxFactory: %s", err)
		}

		fx, err := fxFactory.New(ctx)
		if err != nil {
			return nil, fmt.Errorf("error while creating fx: %s", err)
		}

		// Create the fx
		fxs[i] = &common.Fx{
			ID: fxID,
			Fx: fx,
		}
	}

	consensusParams := m.consensusParams
	consensusParams.Namespace = fmt.Sprintf("gecko_%s", primaryAlias)

	// The validators of this blockchain
	var validators validators.Set // Validators validating this blockchain
	var ok bool
	if m.stakingEnabled {
		validators, ok = m.validators.GetValidatorSet(chainParams.SubnetID)
	} else { // Staking is disabled. Every peer validates every subnet.
		validators, ok = m.validators.GetValidatorSet(ids.Empty) // ids.Empty is the default subnet ID. TODO: Move to const package so we can use it here.
	}
	if !ok {
		return nil, fmt.Errorf("couldn't get validator set of subnet with ID %s. The subnet may not exist", chainParams.SubnetID)
	}

	beacons := validators
	if chainParams.CustomBeacons != nil {
		beacons = chainParams.CustomBeacons
	}

	bootstrapWeight, err := beacons.Weight()
	if err != nil {
		return nil, fmt.Errorf("Error calculating bootstrap weight: %s", err)
	}

	var chain *chain
	switch vm := vm.(type) {
	case avalanche.DAGVM:
		chain, err = m.createAvalancheChain(
			ctx,
			chainParams.GenesisData,
			validators,
			beacons,
			vm,
			fxs,
			consensusParams,
			bootstrapWeight,
		)
		if err != nil {
			return nil, fmt.Errorf("error while creating new avalanche vm %s", err)
		}
	case smeng.ChainVM:
		chain, err = m.createSnowmanChain(
			ctx,
			chainParams.GenesisData,
			validators,
			beacons,
			vm,
			fxs,
			consensusParams.Parameters,
			bootstrapWeight,
		)
		if err != nil {
			return nil, fmt.Errorf("error while creating new snowman vm %s", err)
		}
	default:
		return nil, fmt.Errorf("the vm should have type avalanche.DAGVM or snowman.ChainVM. Chain not created")
	}

	// Allows messages to be routed to the new chain
	m.chainRouter.AddChain(chain.Handler)
	// If the X or P Chain panics, do not attempt to recover
	if chainParams.SubnetID.Equals(ids.Empty) && (chainParams.ID.Equals(ids.Empty) || vmID.Equals(avm.ID)) {
		go ctx.Log.RecoverAndPanic(chain.Handler.Dispatch)
	} else {
		go ctx.Log.RecoverAndExit(chain.Handler.Dispatch, func() {
			ctx.Log.Error("Chain with ID: %s was shutdown due to a panic", chainParams.ID)
		})
	}

	reqWeight := (3*bootstrapWeight + 3) / 4
	if reqWeight == 0 {
		chain.Engine.Startup()
	} else {
		awaiter := NewAwaiter(beacons, reqWeight, func() {
			ctx.Lock.Lock()
			defer ctx.Lock.Unlock()
			chain.Engine.Startup()
		})
		go m.net.RegisterHandler(awaiter)
	}

	return chain, nil
}

// Implements Manager.AddRegistrant
func (m *manager) AddRegistrant(r Registrant) { m.registrants = append(m.registrants, r) }

func (m *manager) unblockChains() {
	m.unblocked = true
	blocked := m.blockedChains
	m.blockedChains = nil
	for _, chainParams := range blocked {
		m.ForceCreateChain(chainParams)
	}
}

// Create a DAG-based blockchain that uses Avalanche
func (m *manager) createAvalancheChain(
	ctx *snow.Context,
	genesisData []byte,
	validators,
	beacons validators.Set,
	vm avalanche.DAGVM,
	fxs []*common.Fx,
	consensusParams avacon.Parameters,
	bootstrapWeight uint64,
) (*chain, error) {
	ctx.Lock.Lock()
	defer ctx.Lock.Unlock()

	db := prefixdb.New(ctx.ChainID.Bytes(), m.db)
	vmDB := prefixdb.New([]byte("vm"), db)
	vertexDB := prefixdb.New([]byte("vertex"), db)
	vertexBootstrappingDB := prefixdb.New([]byte("vertex_bs"), db)
	txBootstrappingDB := prefixdb.New([]byte("tx_bs"), db)

	vtxBlocker, err := queue.New(vertexBootstrappingDB)
	if err != nil {
		return nil, err
	}
	txBlocker, err := queue.New(txBootstrappingDB)
	if err != nil {
		return nil, err
	}

	// The channel through which a VM may send messages to the consensus engine
	// VM uses this channel to notify engine that a block is ready to be made
	msgChan := make(chan common.Message, defaultChannelSize)

	if err := vm.Initialize(ctx, vmDB, genesisData, msgChan, fxs); err != nil {
		return nil, fmt.Errorf("error during vm's Initialize: %w", err)
	}

	// Handles serialization/deserialization of vertices and also the
	// persistence of vertices
	vtxState := &state.Serializer{}
	vtxState.Initialize(ctx, vm, vertexDB)

	// Passes messages from the consensus engine to the network
	sender := sender.Sender{}
	sender.Initialize(ctx, m.net, m.chainRouter, m.timeoutManager)

	// The engine handles consensus
	engine := &avaeng.Transitive{}
	engine.Initialize(avaeng.Config{
		BootstrapConfig: avaeng.BootstrapConfig{
			Config: common.Config{
				Context:    ctx,
				Validators: validators,
				Beacons:    beacons,
				Alpha:      bootstrapWeight/2 + 1, // must be > 50%
				Sender:     &sender,
			},
			VtxBlocked: vtxBlocker,
			TxBlocked:  txBlocker,
			State:      vtxState,
			VM:         vm,
		},
		Params:    consensusParams,
		Consensus: &avacon.Topological{},
	})

	// Asynchronously passes messages from the network to the consensus engine
	handler := &router.Handler{}
	handler.Initialize(
		engine,
		msgChan,
		defaultChannelSize,
		fmt.Sprintf("%s_handler", consensusParams.Namespace),
		consensusParams.Metrics,
	)

	return &chain{
		Engine:  engine,
		Handler: handler,
		VM:      vm,
		Ctx:     ctx,
	}, nil
}

// Create a linear chain using the Snowman consensus engine
func (m *manager) createSnowmanChain(
	ctx *snow.Context,
	genesisData []byte,
	validators,
	beacons validators.Set,
	vm smeng.ChainVM,
	fxs []*common.Fx,
	consensusParams snowball.Parameters,
	bootstrapWeight uint64,
) (*chain, error) {
	ctx.Lock.Lock()
	defer ctx.Lock.Unlock()

	db := prefixdb.New(ctx.ChainID.Bytes(), m.db)
	vmDB := prefixdb.New([]byte("vm"), db)
	bootstrappingDB := prefixdb.New([]byte("bs"), db)

	blocked, err := queue.New(bootstrappingDB)
	if err != nil {
		return nil, err
	}

	// The channel through which a VM may send messages to the consensus engine
	// VM uses this channel to notify engine that a block is ready to be made
	msgChan := make(chan common.Message, defaultChannelSize)

	// Initialize the VM
	if err := vm.Initialize(ctx, vmDB, genesisData, msgChan, fxs); err != nil {
		return nil, err
	}

	// Passes messages from the consensus engine to the network
	sender := sender.Sender{}
	sender.Initialize(ctx, m.net, m.chainRouter, m.timeoutManager)

	// The engine handles consensus
	engine := &smeng.Transitive{}
	engine.Initialize(smeng.Config{
		BootstrapConfig: smeng.BootstrapConfig{
			Config: common.Config{
				Context:    ctx,
				Validators: validators,
				Beacons:    beacons,
				Alpha:      bootstrapWeight/2 + 1, // must be > 50%
				Sender:     &sender,
			},
			Blocked:      blocked,
			VM:           vm,
			Bootstrapped: m.unblockChains,
		},
		Params:    consensusParams,
		Consensus: &smcon.Topological{},
	})

	// Asynchronously passes messages from the network to the consensus engine
	handler := &router.Handler{}
	handler.Initialize(
		engine,
		msgChan,
		defaultChannelSize,
		fmt.Sprintf("%s_handler", consensusParams.Namespace),
		consensusParams.Metrics,
	)

	return &chain{
		Engine:  engine,
		Handler: handler,
		VM:      vm,
		Ctx:     ctx,
	}, nil
}

func (m *manager) IsBootstrapped(id ids.ID) bool {
	chain, exists := m.chains[id.Key()]
	if !exists {
		return false
	}

	return chain.Engine().IsBootstrapped()
}

// Shutdown stops all the chains
func (m *manager) Shutdown() {
	m.chainRouter.Shutdown()
}

// LookupVM returns the ID of the VM associated with an alias
func (m *manager) LookupVM(alias string) (ids.ID, error) { return m.vmManager.Lookup(alias) }

// Notify registrants [those who want to know about the creation of chains]
// that the specified chain has been created
func (m *manager) notifyRegistrants(ctx *snow.Context, vm interface{}) {
	for _, registrant := range m.registrants {
		registrant.RegisterChain(ctx, vm)
	}
}

// Returns:
// 1) the alias that already exists, or the empty string if there is none
// 2) true iff there exists a chain such that the chain has an alias in [aliases]
func (m *manager) isChainWithAlias(aliases ...string) (string, bool) {
	for _, alias := range aliases {
		if _, err := m.Lookup(alias); err == nil {
			return alias, true
		}
	}
	return "", false
}
