package state_native

import (
	"errors"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/state/state-native/types"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/state/stateutil"
	fieldparams "github.com/OffchainLabs/prysm/v7/config/fieldparams"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/encoding/ssz"
	enginev1 "github.com/OffchainLabs/prysm/v7/proto/engine/v1"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/runtime/version"
)

// SetPayloadExpectedWithdrawals updates the cached withdrawals root for the next payload.
func (b *BeaconState) SetPayloadExpectedWithdrawals(withdrawals []*enginev1.Withdrawal) error {
	if b.version < version.Gloas {
		return errNotSupported("SetPayloadExpectedWithdrawals", b.version)
	}

	if withdrawals == nil {
		return errors.New("cannot set nil payload expected withdrawals")
	}

	root, err := ssz.WithdrawalSliceRoot(withdrawals, fieldparams.MaxWithdrawalsPerPayload)
	if err != nil {
		return err
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	rootBytes := make([]byte, len(root))
	copy(rootBytes, root[:])
	b.latestWithdrawalsRoot = rootBytes
	b.markFieldAsDirty(types.LatestWithdrawalsRoot)

	return nil
}

// DequeueBuilderPendingWithdrawals removes processed builder withdrawals from the front of the queue.
func (b *BeaconState) DequeueBuilderPendingWithdrawals(n uint64) error {
	if b.version < version.Gloas {
		return errNotSupported("DequeueBuilderPendingWithdrawals", b.version)
	}

	if n > uint64(len(b.builderPendingWithdrawals)) {
		return errors.New("cannot dequeue more builder withdrawals than are in the queue")
	}

	if n == 0 {
		return nil
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	if b.sharedFieldReferences[types.BuilderPendingWithdrawals].Refs() > 1 {
		withdrawals := make([]*ethpb.BuilderPendingWithdrawal, len(b.builderPendingWithdrawals))
		copy(withdrawals, b.builderPendingWithdrawals)
		b.builderPendingWithdrawals = withdrawals
		b.sharedFieldReferences[types.BuilderPendingWithdrawals].MinusRef()
		b.sharedFieldReferences[types.BuilderPendingWithdrawals] = stateutil.NewRef(1)
	}

	b.builderPendingWithdrawals = b.builderPendingWithdrawals[n:]
	b.markFieldAsDirty(types.BuilderPendingWithdrawals)
	b.rebuildTrie[types.BuilderPendingWithdrawals] = true

	return nil
}

// SetNextWithdrawalBuilderIndex sets the next builder index for the withdrawals sweep.
func (b *BeaconState) SetNextWithdrawalBuilderIndex(_ primitives.BuilderIndex) error {
	if b.version < version.Gloas {
		return errNotSupported("SetNextWithdrawalBuilderIndex", b.version)
	}

	return nil
}
