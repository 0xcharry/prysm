package gloas

import (
	"testing"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/helpers"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/state"
	state_native "github.com/OffchainLabs/prysm/v7/beacon-chain/state/state-native"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/testing/require"
)

func TestBuilderQuorumThreshold(t *testing.T) {
	helpers.ClearCache()
	cfg := params.BeaconConfig()

	validators := []*ethpb.Validator{
		{EffectiveBalance: cfg.MaxEffectiveBalance, ActivationEpoch: 0, ExitEpoch: 1},
		{EffectiveBalance: cfg.MaxEffectiveBalance, ActivationEpoch: 0, ExitEpoch: 1},
	}
	st, err := state_native.InitializeFromProtoUnsafeGloas(&ethpb.BeaconStateGloas{Validators: validators})
	require.NoError(t, err)

	got, err := builderQuorumThreshold(st)
	require.NoError(t, err)

	total := uint64(len(validators)) * cfg.MaxEffectiveBalance
	perSlot := total / uint64(cfg.SlotsPerEpoch)
	want := (perSlot * cfg.BuilderPaymentThresholdNumerator) / cfg.BuilderPaymentThresholdDenominator
	require.Equal(t, primitives.Gwei(want), got)
}

func TestProcessBuilderPendingPayments(t *testing.T) {
	helpers.ClearCache()
	cfg := params.BeaconConfig()

	validators := []*ethpb.Validator{
		{EffectiveBalance: cfg.MaxEffectiveBalance, ActivationEpoch: 0, ExitEpoch: 1},
		{EffectiveBalance: cfg.MaxEffectiveBalance, ActivationEpoch: 0, ExitEpoch: 1},
	}
	pbSt, err := state_native.InitializeFromProtoPhase0(&ethpb.BeaconState{Validators: validators})
	require.NoError(t, err)

	total := uint64(len(validators)) * cfg.MaxEffectiveBalance
	perSlot := total / uint64(cfg.SlotsPerEpoch)
	quorum := (perSlot * cfg.BuilderPaymentThresholdNumerator) / cfg.BuilderPaymentThresholdDenominator

	slotsPerEpoch := int(cfg.SlotsPerEpoch)
	payments := make([]*ethpb.BuilderPendingPayment, 2*slotsPerEpoch)
	for i := range payments {
		payments[i] = &ethpb.BuilderPendingPayment{
			Withdrawal: &ethpb.BuilderPendingWithdrawal{
				FeeRecipient: make([]byte, 20),
			},
		}
	}
	payments[0].Weight = primitives.Gwei(quorum + 1)
	payments[0].Withdrawal.Amount = 1

	st := &testProcessState{
		BeaconState: pbSt,
		payments:    payments,
		exitEpoch:   7,
	}
	rotatedHead := st.payments[slotsPerEpoch]

	require.NoError(t, ProcessBuilderPendingPayments(st))
	require.Equal(t, 1, len(st.withdrawals))

	wantEpoch, err := st.exitEpoch.SafeAdd(uint64(cfg.MinValidatorWithdrawabilityDelay))
	require.NoError(t, err)
	require.Equal(t, wantEpoch, st.withdrawals[0].WithdrawableEpoch)
	require.Equal(t, rotatedHead, st.payments[0])
}

type testProcessState struct {
	state.BeaconState
	payments    []*ethpb.BuilderPendingPayment
	withdrawals []*ethpb.BuilderPendingWithdrawal
	exitEpoch   primitives.Epoch
}

func (t *testProcessState) BuilderPendingPayments() ([]*ethpb.BuilderPendingPayment, error) {
	return t.payments, nil
}

func (t *testProcessState) AppendBuilderPendingWithdrawal(withdrawal *ethpb.BuilderPendingWithdrawal) error {
	t.withdrawals = append(t.withdrawals, withdrawal)
	return nil
}

func (t *testProcessState) RotateBuilderPendingPayments() error {
	slotsPerEpoch := int(params.BeaconConfig().SlotsPerEpoch)
	rotated := append([]*ethpb.BuilderPendingPayment{}, t.payments[slotsPerEpoch:]...)
	for i := 0; i < slotsPerEpoch; i++ {
		rotated = append(rotated, &ethpb.BuilderPendingPayment{
			Withdrawal: &ethpb.BuilderPendingWithdrawal{
				FeeRecipient: make([]byte, 20),
			},
		})
	}
	t.payments = rotated
	return nil
}

func (t *testProcessState) ExitEpochAndUpdateChurn(_ primitives.Gwei) (primitives.Epoch, error) {
	return t.exitEpoch, nil
}
