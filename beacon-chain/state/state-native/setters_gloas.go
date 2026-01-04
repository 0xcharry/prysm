package state_native

import (
	"github.com/OffchainLabs/prysm/v7/beacon-chain/state/state-native/types"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
)

// SetBuilderPendingPayment sets a builder pending payment for the specified slot.
func (b *BeaconState) SetBuilderPendingPayment(index int, payment *ethpb.BuilderPendingPayment) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.builderPendingPayments[index] = payment
	b.markFieldAsDirty(types.BuilderPendingPayments)
	return nil
}
