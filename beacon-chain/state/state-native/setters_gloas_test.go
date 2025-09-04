package state_native

import (
	"testing"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/state/state-native/types"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/state/stateutil"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/testing/require"
)

func TestRotateBuilderPendingPayments(t *testing.T) {
	totalPayments := 2 * params.BeaconConfig().SlotsPerEpoch
	payments := make([]*ethpb.BuilderPendingPayment, totalPayments)
	for i := range payments {
		idx := uint64(i)
		payments[i] = &ethpb.BuilderPendingPayment{
			Weight: primitives.Gwei(idx * 100e9),
			Withdrawal: &ethpb.BuilderPendingWithdrawal{
				FeeRecipient:      make([]byte, 20),
				Amount:            primitives.Gwei(idx * 1e9),
				BuilderIndex:      primitives.ValidatorIndex(idx + 100),
				WithdrawableEpoch: primitives.Epoch(idx + 1000),
			},
		}
	}

	statePb, err := InitializeFromProtoUnsafeGloas(&ethpb.BeaconStateGloas{
		BuilderPendingPayments: payments,
	})
	require.NoError(t, err)
	st, ok := statePb.(*BeaconState)
	require.Equal(t, true, ok)
	st.sharedFieldReferences[types.BuilderPendingPayments] = stateutil.NewRef(1)

	oldPayments, err := st.BuilderPendingPayments()
	require.NoError(t, err)
	require.NoError(t, st.RotateBuilderPendingPayments())

	newPayments, err := st.BuilderPendingPayments()
	require.NoError(t, err)
	slotsPerEpoch := int(params.BeaconConfig().SlotsPerEpoch)
	for i := 0; i < slotsPerEpoch; i++ {
		require.DeepEqual(t, oldPayments[slotsPerEpoch+i], newPayments[i])
	}

	for i := slotsPerEpoch; i < 2*slotsPerEpoch; i++ {
		payment := newPayments[i]
		require.Equal(t, primitives.Gwei(0), payment.Weight)
		require.Equal(t, 20, len(payment.Withdrawal.FeeRecipient))
		require.Equal(t, primitives.Gwei(0), payment.Withdrawal.Amount)
		require.Equal(t, primitives.ValidatorIndex(0), payment.Withdrawal.BuilderIndex)
		require.Equal(t, primitives.Epoch(0), payment.Withdrawal.WithdrawableEpoch)
	}
}

func TestAppendBuilderPendingWithdrawal_CopyOnWrite(t *testing.T) {
	wd := &ethpb.BuilderPendingWithdrawal{
		FeeRecipient:      make([]byte, 20),
		Amount:            1,
		BuilderIndex:      2,
		WithdrawableEpoch: 3,
	}
	statePb, err := InitializeFromProtoUnsafeGloas(&ethpb.BeaconStateGloas{
		BuilderPendingWithdrawals: []*ethpb.BuilderPendingWithdrawal{wd},
	})
	require.NoError(t, err)

	st, ok := statePb.(*BeaconState)
	require.Equal(t, true, ok)

	copied := st.Copy().(*BeaconState)
	require.Equal(t, uint(2), st.sharedFieldReferences[types.BuilderPendingWithdrawals].Refs())

	appended := &ethpb.BuilderPendingWithdrawal{
		FeeRecipient:      make([]byte, 20),
		Amount:            4,
		BuilderIndex:      5,
		WithdrawableEpoch: 6,
	}
	require.NoError(t, copied.AppendBuilderPendingWithdrawal(appended))

	require.Equal(t, 1, len(st.builderPendingWithdrawals))
	require.Equal(t, 2, len(copied.builderPendingWithdrawals))
	require.DeepEqual(t, wd, copied.builderPendingWithdrawals[0])
	require.DeepEqual(t, appended, copied.builderPendingWithdrawals[1])
	require.DeepEqual(t, wd, st.builderPendingWithdrawals[0])
	require.Equal(t, uint(1), st.sharedFieldReferences[types.BuilderPendingWithdrawals].Refs())
	require.Equal(t, uint(1), copied.sharedFieldReferences[types.BuilderPendingWithdrawals].Refs())
}
