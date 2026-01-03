package state_native

import (
	"github.com/OffchainLabs/prysm/v7/beacon-chain/state/state-native/types"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/interfaces"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
)

func (b *BeaconState) SetExecutionPayloadBid(h interfaces.ROExecutionPayloadBid) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	parentBlockHash := h.ParentBlockHash()
	parentBlockRoot := h.ParentBlockRoot()
	blockHash := h.BlockHash()
	randao := h.PrevRandao()
	blobKzgCommitmentsRoot := h.BlobKzgCommitmentsRoot()
	feeRecipient := h.FeeRecipient()
	b.latestExecutionPayloadBid = &ethpb.ExecutionPayloadBid{
		ParentBlockHash:        parentBlockHash[:],
		ParentBlockRoot:        parentBlockRoot[:],
		BlockHash:              blockHash[:],
		PrevRandao:             randao[:],
		GasLimit:               h.GasLimit(),
		BuilderIndex:           h.BuilderIndex(),
		Slot:                   h.Slot(),
		Value:                  h.Value(),
		ExecutionPayment:       h.ExecutionPayment(),
		BlobKzgCommitmentsRoot: blobKzgCommitmentsRoot[:],
		FeeRecipient:           feeRecipient[:],
	}
	b.markFieldAsDirty(types.LatestExecutionPayloadBid)

	return nil
}

// SetBuilderPendingPayment sets a builder pending payment for the specified slot.
func (b *BeaconState) SetBuilderPendingPayment(slot primitives.Slot, payment *ethpb.BuilderPendingPayment) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	paymentIndex := slotsPerEpoch + (slot % slotsPerEpoch)

	b.builderPendingPayments[paymentIndex] = ethpb.CopyBuilderPendingPayment(payment)

	b.markFieldAsDirty(types.BuilderPendingPayments)
	return nil
}

// SetLatestBlockHash sets the latest execution block hash.
func (b *BeaconState) SetLatestBlockHash(hash [32]byte) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.latestBlockHash = hash[:]
	b.markFieldAsDirty(types.LatestBlockHash)
	return nil
}

// SetExecutionPayloadAvailability sets the execution payload availability bit for a specific slot.
func (b *BeaconState) SetExecutionPayloadAvailability(index primitives.Slot, available bool) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	bitIndex := index % params.BeaconConfig().SlotsPerHistoricalRoot
	byteIndex := bitIndex / 8
	bitPosition := bitIndex % 8

	// Set or clear the bit
	if available {
		b.executionPayloadAvailability[byteIndex] |= 1 << bitPosition
	} else {
		b.executionPayloadAvailability[byteIndex] &^= 1 << bitPosition
	}

	b.markFieldAsDirty(types.ExecutionPayloadAvailability)
	return nil
}

// SetBuilderPendingPayments sets the entire builder pending payments array.
func (b *BeaconState) SetBuilderPendingPayments(payments []*ethpb.BuilderPendingPayment) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.builderPendingPayments = ethpb.CopyBuilderPendingPaymentSlice(payments)
	b.markFieldAsDirty(types.BuilderPendingPayments)
	return nil
}

// AppendBuilderPendingWithdrawal appends a builder pending withdrawal to the list.
func (b *BeaconState) AppendBuilderPendingWithdrawal(withdrawal *ethpb.BuilderPendingWithdrawal) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.builderPendingWithdrawals = append(b.builderPendingWithdrawals, ethpb.CopyBuilderPendingWithdrawal(withdrawal))
	b.markFieldAsDirty(types.BuilderPendingWithdrawals)
	return nil
}
