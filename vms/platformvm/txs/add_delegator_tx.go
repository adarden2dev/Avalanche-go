// Copyright (C) 2019-2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txs

import (
	"errors"
	"fmt"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/math"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/components/verify"
	"github.com/ava-labs/avalanchego/vms/platformvm/fx"
	"github.com/ava-labs/avalanchego/vms/platformvm/validator"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
)

var (
	_ UnsignedTx             = &AddDelegatorTx{}
	_ StakerTx               = &AddDelegatorTx{}
	_ secp256k1fx.UnsignedTx = &AddDelegatorTx{}

	errDelegatorWeightMismatch = errors.New("delegator weight is not equal to total stake weight")
)

// AddDelegatorTx is an unsigned addDelegatorTx
type AddDelegatorTx struct {
	// Metadata, inputs and outputs
	BaseTx `serialize:"true"`
	// Describes the delegatee
	Validator validator.Validator `serialize:"true" json:"validator"`
	// Where to send staked tokens when done validating
	Stake []*avax.TransferableOutput `serialize:"true" json:"stake"`
	// Where to send staking rewards when done validating
	RewardsOwner fx.Owner `serialize:"true" json:"rewardsOwner"`
}

// InitCtx sets the FxID fields in the inputs and outputs of this
// [UnsignedAddDelegatorTx]. Also sets the [ctx] to the given [vm.ctx] so that
// the addresses can be json marshalled into human readable format
func (tx *AddDelegatorTx) InitCtx(ctx *snow.Context) {
	tx.BaseTx.InitCtx(ctx)
	for _, out := range tx.Stake {
		out.FxID = secp256k1fx.ID
		out.InitCtx(ctx)
	}
	tx.RewardsOwner.InitCtx(ctx)
}

func (tx *AddDelegatorTx) SubnetID() ids.ID     { return constants.PrimaryNetworkID }
func (tx *AddDelegatorTx) NodeID() ids.NodeID   { return tx.Validator.NodeID }
func (tx *AddDelegatorTx) StartTime() time.Time { return tx.Validator.StartTime() }
func (tx *AddDelegatorTx) EndTime() time.Time   { return tx.Validator.EndTime() }
func (tx *AddDelegatorTx) Weight() uint64       { return tx.Validator.Wght }
func (tx *AddDelegatorTx) PendingPriority() Priority {
	return PrimaryNetworkDelegatorApricotPendingPriority
}
func (tx *AddDelegatorTx) CurrentPriority() Priority { return PrimaryNetworkDelegatorCurrentPriority }

// SyntacticVerify returns nil iff [tx] is valid
func (tx *AddDelegatorTx) SyntacticVerify(ctx *snow.Context) error {
	switch {
	case tx == nil:
		return ErrNilTx
	case tx.SyntacticallyVerified: // already passed syntactic verification
		return nil
	}

	if err := tx.BaseTx.SyntacticVerify(ctx); err != nil {
		return err
	}
	if err := verify.All(&tx.Validator, tx.RewardsOwner); err != nil {
		return fmt.Errorf("failed to verify validator or rewards owner: %w", err)
	}

	totalStakeWeight := uint64(0)
	for _, out := range tx.Stake {
		if err := out.Verify(); err != nil {
			return fmt.Errorf("output verification failed: %w", err)
		}
		newWeight, err := math.Add64(totalStakeWeight, out.Output().Amount())
		if err != nil {
			return err
		}
		totalStakeWeight = newWeight

		assetID := out.AssetID()
		if assetID != ctx.AVAXAssetID {
			return fmt.Errorf("stake output must be AVAX but is %q", assetID)
		}
	}

	switch {
	case !avax.IsSortedTransferableOutputs(tx.Stake, Codec):
		return errOutputsNotSorted
	case totalStakeWeight != tx.Validator.Wght:
		return fmt.Errorf("%w, delegator weight %d total stake weight %d",
			errDelegatorWeightMismatch,
			tx.Validator.Wght,
			totalStakeWeight,
		)
	}

	// cache that this is valid
	tx.SyntacticallyVerified = true
	return nil
}

func (tx *AddDelegatorTx) Visit(visitor Visitor) error {
	return visitor.AddDelegatorTx(tx)
}
