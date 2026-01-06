package gloas

import (
	"fmt"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/state"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/encoding/bytesutil"
	enginev1 "github.com/OffchainLabs/prysm/v7/proto/engine/v1"
	"github.com/OffchainLabs/prysm/v7/time/slots"
)

// getBuilderWithdrawals returns builder withdrawals derived from pending builder withdrawals.
// Spec v1.6.1 (pseudocode):
// def get_builder_withdrawals(state, withdrawal_index, prior_withdrawals):
//
//	withdrawals_limit = MAX_WITHDRAWALS_PER_PAYLOAD
//
//	processed_count = 0
//	withdrawals = []
//	for withdrawal in state.builder_pending_withdrawals:
//	    all_withdrawals = prior_withdrawals + withdrawals
//	    has_reached_limit = len(all_withdrawals) == withdrawals_limit
//	    if has_reached_limit:
//	        break
//
//	    builder_index = withdrawal.builder_index
//	    withdrawals.append(
//	        Withdrawal(
//	            index=withdrawal_index,
//	            validator_index=convert_builder_index_to_validator_index(builder_index),
//	            address=withdrawal.fee_recipient,
//	            amount=withdrawal.amount,
//	        )
//	    )
//	    withdrawal_index += WithdrawalIndex(1)
//	    processed_count += 1
//
//	return withdrawals, withdrawal_index, processed_count
func getBuilderWithdrawals(
	st state.ReadOnlyBeaconState,
	withdrawalIndex uint64,
) ([]*enginev1.Withdrawal, uint64, uint64, error) {
	pendingWithdrawals, err := st.BuilderPendingWithdrawals()
	if err != nil {
		return nil, withdrawalIndex, 0, err
	}

	withdrawalsLimit := params.BeaconConfig().MaxWithdrawalsPerPayload
	withdrawals := make([]*enginev1.Withdrawal, 0, len(pendingWithdrawals))
	var processedCount uint64
	for _, withdrawal := range pendingWithdrawals {
		if uint64(+len(withdrawals)) == withdrawalsLimit {
			break
		}

		withdrawals = append(withdrawals, &enginev1.Withdrawal{
			Index:          withdrawalIndex,
			ValidatorIndex: ConvertBuilderIndexToValidatorIndex(primitives.BuilderIndex(withdrawal.BuilderIndex)),
			Address:        bytesutil.SafeCopyBytes(withdrawal.FeeRecipient),
			Amount:         uint64(withdrawal.Amount),
		})
		withdrawalIndex++
		processedCount++
	}

	return withdrawals, withdrawalIndex, processedCount, nil
}

// getBuildersSweepWithdrawals returns builder withdrawals selected by a sweep over the builders registry.
// Spec v1.6.1 (pseudocode):
// def get_builders_sweep_withdrawals(state, withdrawal_index, prior_withdrawals):
//
//	epoch = get_current_epoch(state)
//	builders_limit = min(len(state.builders), MAX_BUILDERS_PER_WITHDRAWALS_SWEEP)
//	withdrawals_limit = MAX_WITHDRAWALS_PER_PAYLOAD
//
//	processed_count = 0
//	withdrawals = []
//	builder_index = state.next_withdrawal_builder_index
//	for _ in range(builders_limit):
//	    all_withdrawals = prior_withdrawals + withdrawals
//	    has_reached_limit = len(all_withdrawals) == withdrawals_limit
//	    if has_reached_limit:
//	        break
//
//	    builder = state.builders[builder_index]
//	    if builder.withdrawable_epoch <= epoch and builder.balance > 0:
//	        withdrawals.append(
//	            Withdrawal(
//	                index=withdrawal_index,
//	                validator_index=convert_builder_index_to_validator_index(builder_index),
//	                address=builder.execution_address,
//	                amount=builder.balance,
//	            )
//	        )
//	        withdrawal_index += WithdrawalIndex(1)
//
//	    builder_index = BuilderIndex((builder_index + 1) % len(state.builders))
//	    processed_count += 1
//
//	return withdrawals, withdrawal_index, processed_count
func getBuildersSweepWithdrawals(
	st state.ReadOnlyBeaconState,
	withdrawalIndex uint64,
	priorWithdrawals []*enginev1.Withdrawal,
) ([]*enginev1.Withdrawal, uint64, uint64, error) {
	builders, err := st.Builders()
	if err != nil {
		return nil, withdrawalIndex, 0, err
	}
	if len(builders) == 0 {
		return nil, withdrawalIndex, 0, nil
	}

	builderIndex, err := st.NextWithdrawalBuilderIndex()
	if err != nil {
		return nil, withdrawalIndex, 0, err
	}
	if uint64(builderIndex) >= uint64(len(builders)) {
		return nil, withdrawalIndex, 0, fmt.Errorf("next withdrawal builder index %d out of range", builderIndex)
	}

	cfg := params.BeaconConfig()
	buildersLimit := len(builders)
	if maxBuilders := int(cfg.MaxBuildersPerWithdrawalsSweep); buildersLimit > maxBuilders {
		buildersLimit = maxBuilders
	}
	withdrawalsLimit := cfg.MaxWithdrawalsPerPayload
	epoch := slots.ToEpoch(st.Slot())

	withdrawals := make([]*enginev1.Withdrawal, 0, buildersLimit)
	var processedCount uint64
	for i := 0; i < buildersLimit; i++ {
		if uint64(len(priorWithdrawals)+len(withdrawals)) == withdrawalsLimit {
			break
		}

		builder := builders[builderIndex]
		if builder != nil && builder.WithdrawableEpoch <= epoch && builder.Balance > 0 {
			withdrawals = append(withdrawals, &enginev1.Withdrawal{
				Index:          withdrawalIndex,
				ValidatorIndex: ConvertBuilderIndexToValidatorIndex(builderIndex),
				Address:        bytesutil.SafeCopyBytes(builder.ExecutionAddress),
				Amount:         uint64(builder.Balance),
			})
			withdrawalIndex++
		}

		builderIndex = primitives.BuilderIndex((uint64(builderIndex) + 1) % uint64(len(builders)))
		processedCount++
	}

	return withdrawals, withdrawalIndex, processedCount, nil
}

// updatePayloadExpectedWithdrawals stores the expected withdrawals for the next payload.
// Spec v1.6.1 (pseudocode):
// def update_payload_expected_withdrawals(state, withdrawals):
//
//	state.payload_expected_withdrawals = List[Withdrawal, MAX_WITHDRAWALS_PER_PAYLOAD](withdrawals)
func updatePayloadExpectedWithdrawals(st state.BeaconState, withdrawals []*enginev1.Withdrawal) error {
	return st.SetPayloadExpectedWithdrawals(withdrawals)
}

// updateBuilderPendingWithdrawals removes processed builder pending withdrawals.
// Spec v1.6.1 (pseudocode):
// def update_builder_pending_withdrawals(state, processed_builder_withdrawals_count):
//
//	state.builder_pending_withdrawals = state.builder_pending_withdrawals[processed_builder_withdrawals_count:]
func updateBuilderPendingWithdrawals(st state.BeaconState, processedBuilderWithdrawalsCount uint64) error {
	if processedBuilderWithdrawalsCount == 0 {
		return nil
	}

	return st.DequeueBuilderPendingWithdrawals(processedBuilderWithdrawalsCount)
}

// updateNextWithdrawalBuilderIndex advances the sweep position for builders.
// Spec v1.6.1 (pseudocode):
// def update_next_withdrawal_builder_index(state, processed_builders_sweep_count):
//
//	if len(state.builders) > 0:
//	    next_index = state.next_withdrawal_builder_index + processed_builders_sweep_count
//	    next_builder_index = BuilderIndex(next_index % len(state.builders))
//	    state.next_withdrawal_builder_index = next_builder_index
func updateNextWithdrawalBuilderIndex(st state.BeaconState, processedBuildersSweepCount uint64) error {
	buildersCount := st.BuildersCount()
	if buildersCount == 0 {
		return nil
	}

	nextIndex, err := st.NextWithdrawalBuilderIndex()
	if err != nil {
		return err
	}

	nextBuilderIndex := primitives.BuilderIndex((uint64(nextIndex) + processedBuildersSweepCount) % uint64(buildersCount))
	return st.SetNextWithdrawalBuilderIndex(nextBuilderIndex)
}
