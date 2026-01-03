package state_native

import (
	"testing"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/state/state-native/types"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/testing/require"
)

func TestSetBuilderPendingPayment(t *testing.T) {
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	slot := primitives.Slot(7)
	paymentIndex := slotsPerEpoch + (slot % slotsPerEpoch)
	payment := &ethpb.BuilderPendingPayment{
		Weight: 11,
		Withdrawal: &ethpb.BuilderPendingWithdrawal{
			FeeRecipient:      []byte{1, 2, 3, 4, 5},
			Amount:            99,
			BuilderIndex:      3,
			WithdrawableEpoch: 5,
		},
	}

	state := &BeaconState{
		builderPendingPayments: make([]*ethpb.BuilderPendingPayment, slotsPerEpoch*2),
		dirtyFields:            make(map[types.FieldIndex]bool),
	}

	require.NoError(t, state.SetBuilderPendingPayment(slot, payment))
	require.Equal(t, true, state.dirtyFields[types.BuilderPendingPayments])
	require.DeepEqual(t, payment, state.builderPendingPayments[paymentIndex])

	state.builderPendingPayments[paymentIndex].Withdrawal.FeeRecipient[0] = 9
	require.Equal(t, byte(1), payment.Withdrawal.FeeRecipient[0])
}

func TestSetLatestBlockHash(t *testing.T) {
	var hash [32]byte
	copy(hash[:], []byte("latest-block-hash"))

	state := &BeaconState{
		dirtyFields: make(map[types.FieldIndex]bool),
	}

	require.NoError(t, state.SetLatestBlockHash(hash))
	require.Equal(t, true, state.dirtyFields[types.LatestBlockHash])
	require.DeepEqual(t, hash[:], state.latestBlockHash)
}

func TestSetExecutionPayloadAvailability(t *testing.T) {
	state := &BeaconState{
		executionPayloadAvailability: make([]byte, params.BeaconConfig().SlotsPerHistoricalRoot/8),
		dirtyFields:                  make(map[types.FieldIndex]bool),
	}

	slot := primitives.Slot(10)
	bitIndex := slot % params.BeaconConfig().SlotsPerHistoricalRoot
	byteIndex := bitIndex / 8
	bitPosition := bitIndex % 8

	require.NoError(t, state.SetExecutionPayloadAvailability(slot, true))
	require.Equal(t, true, state.dirtyFields[types.ExecutionPayloadAvailability])
	require.Equal(t, byte(1<<bitPosition), state.executionPayloadAvailability[byteIndex]&(1<<bitPosition))

	require.NoError(t, state.SetExecutionPayloadAvailability(slot, false))
	require.Equal(t, byte(0), state.executionPayloadAvailability[byteIndex]&(1<<bitPosition))
}

func TestSetBuilderPendingPayments(t *testing.T) {
	payments := []*ethpb.BuilderPendingPayment{
		{
			Weight: 1,
			Withdrawal: &ethpb.BuilderPendingWithdrawal{
				FeeRecipient:      []byte{1, 1, 1},
				Amount:            5,
				BuilderIndex:      2,
				WithdrawableEpoch: 3,
			},
		},
		{
			Weight: 2,
			Withdrawal: &ethpb.BuilderPendingWithdrawal{
				FeeRecipient:      []byte{2, 2, 2},
				Amount:            6,
				BuilderIndex:      4,
				WithdrawableEpoch: 7,
			},
		},
	}

	state := &BeaconState{
		dirtyFields: make(map[types.FieldIndex]bool),
	}

	require.NoError(t, state.SetBuilderPendingPayments(payments))
	require.Equal(t, true, state.dirtyFields[types.BuilderPendingPayments])
	require.DeepEqual(t, payments, state.builderPendingPayments)

	state.builderPendingPayments[0].Weight = 99
	require.Equal(t, primitives.Gwei(1), payments[0].Weight)
}

func TestAppendBuilderPendingWithdrawal(t *testing.T) {
	withdrawal := &ethpb.BuilderPendingWithdrawal{
		FeeRecipient:      []byte{9, 8, 7},
		Amount:            77,
		BuilderIndex:      12,
		WithdrawableEpoch: 13,
	}

	state := &BeaconState{
		dirtyFields: make(map[types.FieldIndex]bool),
	}

	require.NoError(t, state.AppendBuilderPendingWithdrawal(withdrawal))
	require.Equal(t, true, state.dirtyFields[types.BuilderPendingWithdrawals])
	require.Equal(t, 1, len(state.builderPendingWithdrawals))
	require.DeepEqual(t, withdrawal, state.builderPendingWithdrawals[0])

	state.builderPendingWithdrawals[0].FeeRecipient[0] = 1
	require.Equal(t, byte(9), withdrawal.FeeRecipient[0])
}
