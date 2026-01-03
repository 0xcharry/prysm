package state_native_test

import (
	"testing"

	state_native "github.com/OffchainLabs/prysm/v7/beacon-chain/state/state-native"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/testing/assert"
	"github.com/OffchainLabs/prysm/v7/testing/require"
)

func TestLatestWithdrawalsRoot(t *testing.T) {
	t.Run("nil root returns zero", func(t *testing.T) {
		s, err := state_native.InitializeFromProtoGloas(&ethpb.BeaconStateGloas{})
		require.NoError(t, err)

		root, err := s.LatestWithdrawalsRoot()
		require.NoError(t, err)
		assert.Equal(t, [32]byte{}, root)
	})

	t.Run("returns value", func(t *testing.T) {
		var want [32]byte
		copy(want[:], []byte("withdrawals-root"))

		s, err := state_native.InitializeFromProtoGloas(&ethpb.BeaconStateGloas{
			LatestWithdrawalsRoot: want[:],
		})
		require.NoError(t, err)

		root, err := s.LatestWithdrawalsRoot()
		require.NoError(t, err)
		assert.Equal(t, want, root)
	})
}

func TestLatestBlockHash(t *testing.T) {
	t.Run("nil hash returns zero", func(t *testing.T) {
		s, err := state_native.InitializeFromProtoGloas(&ethpb.BeaconStateGloas{})
		require.NoError(t, err)

		hash, err := s.LatestBlockHash()
		require.NoError(t, err)
		assert.Equal(t, [32]byte{}, hash)
	})

	t.Run("returns value", func(t *testing.T) {
		var want [32]byte
		copy(want[:], []byte("latest-block-hash"))

		s, err := state_native.InitializeFromProtoGloas(&ethpb.BeaconStateGloas{
			LatestBlockHash: want[:],
		})
		require.NoError(t, err)

		hash, err := s.LatestBlockHash()
		require.NoError(t, err)
		assert.Equal(t, want, hash)
	})
}

func TestBuilderPendingPayment(t *testing.T) {
	t.Run("returns payment for slot", func(t *testing.T) {
		slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
		slot := primitives.Slot(7)
		paymentIndex := slotsPerEpoch + (slot % slotsPerEpoch)
		payments := make([]*ethpb.BuilderPendingPayment, slotsPerEpoch*2)
		payment := &ethpb.BuilderPendingPayment{
			Weight: 11,
			Withdrawal: &ethpb.BuilderPendingWithdrawal{
				FeeRecipient:      []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
				Amount:            99,
				BuilderIndex:      3,
				WithdrawableEpoch: 5,
			},
		}
		payments[paymentIndex] = payment

		s, err := state_native.InitializeFromProtoGloas(&ethpb.BeaconStateGloas{
			BuilderPendingPayments: payments,
		})
		require.NoError(t, err)

		got, err := s.BuilderPendingPayment(slot)
		require.NoError(t, err)
		require.DeepEqual(t, payment, got)

		got.Withdrawal.FeeRecipient[0] = 9
		require.Equal(t, byte(1), payment.Withdrawal.FeeRecipient[0])
	})
}
