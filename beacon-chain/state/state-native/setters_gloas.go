package state_native

import (
	"errors"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/state/state-native/types"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/state/stateutil"
	"github.com/OffchainLabs/prysm/v7/config/params"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
)

// RotateBuilderPendingPayments rotates the queue by dropping slots per epoch payments from the
// front and appending slots per epoch empty payments to the end.
// This implements: state.builder_pending_payments = state.builder_pending_payments[SLOTS_PER_EPOCH:] + [BuilderPendingPayment() for _ in range(SLOTS_PER_EPOCH)]
func (b *BeaconState) RotateBuilderPendingPayments() error {
	b.lock.Lock()
	defer b.lock.Unlock()

	oldPayments := b.builderPendingPayments
	slotsPerEpoch := uint64(params.BeaconConfig().SlotsPerEpoch)
	newPayments := make([]*ethpb.BuilderPendingPayment, 2*slotsPerEpoch)

	copy(newPayments, oldPayments[slotsPerEpoch:])

	for i := uint64(len(oldPayments)) - slotsPerEpoch; i < uint64(len(newPayments)); i++ {
		newPayments[i] = emptyPayment()
	}

	if b.sharedFieldReferences[types.BuilderPendingPayments].Refs() > 1 {
		b.sharedFieldReferences[types.BuilderPendingPayments].MinusRef()
		b.sharedFieldReferences[types.BuilderPendingPayments] = stateutil.NewRef(1)
	}

	b.builderPendingPayments = newPayments
	b.markFieldAsDirty(types.BuilderPendingPayments)
	b.rebuildTrie[types.BuilderPendingPayments] = true
	return nil
}

// AppendBuilderPendingWithdrawal appends a builder pending withdrawal to the beacon state.
// If the withdrawals slice is shared, it copies the slice first to preserve references.
func (b *BeaconState) AppendBuilderPendingWithdrawal(withdrawal *ethpb.BuilderPendingWithdrawal) error {
	if withdrawal == nil {
		return errors.New("cannot append nil builder pending withdrawal")
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	withdrawals := b.builderPendingWithdrawals
	if b.sharedFieldReferences[types.BuilderPendingWithdrawals].Refs() > 1 {
		withdrawals = make([]*ethpb.BuilderPendingWithdrawal, len(b.builderPendingWithdrawals), len(b.builderPendingWithdrawals)+1)
		copy(withdrawals, b.builderPendingWithdrawals)
		b.sharedFieldReferences[types.BuilderPendingWithdrawals].MinusRef()
		b.sharedFieldReferences[types.BuilderPendingWithdrawals] = stateutil.NewRef(1)
	}

	b.builderPendingWithdrawals = append(withdrawals, withdrawal)
	b.markFieldAsDirty(types.BuilderPendingWithdrawals)
	return nil
}

func emptyPayment() *ethpb.BuilderPendingPayment {
	return &ethpb.BuilderPendingPayment{
		Weight: 0,
		Withdrawal: &ethpb.BuilderPendingWithdrawal{
			FeeRecipient:      make([]byte, 20),
			Amount:            0,
			BuilderIndex:      0,
			WithdrawableEpoch: 0,
		},
	}
}
