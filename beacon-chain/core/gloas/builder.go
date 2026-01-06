package gloas

import (
	"github.com/OffchainLabs/prysm/v7/beacon-chain/state"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
)

// IsBuilderIndex returns true when the BuilderIndex flag is set on a validator index.
// Spec v1.6.1 (pseudocode):
// def is_builder_index(validator_index: ValidatorIndex) -> bool:
//     return (validator_index & BUILDER_INDEX_FLAG) != 0
func IsBuilderIndex(validatorIndex primitives.ValidatorIndex) bool {
	return uint64(validatorIndex)&params.BeaconConfig().BuilderIndexFlag != 0
}

// ConvertValidatorIndexToBuilderIndex strips the builder flag from a validator index.
// Spec v1.6.1 (pseudocode):
// def convert_validator_index_to_builder_index(validator_index: ValidatorIndex) -> BuilderIndex:
//     return BuilderIndex(validator_index & ~BUILDER_INDEX_FLAG)
func ConvertValidatorIndexToBuilderIndex(validatorIndex primitives.ValidatorIndex) primitives.BuilderIndex {
	return primitives.BuilderIndex(uint64(validatorIndex) & ^params.BeaconConfig().BuilderIndexFlag)
}

// ConvertBuilderIndexToValidatorIndex sets the builder flag on a builder index.
// Spec v1.6.1 (pseudocode):
// def convert_builder_index_to_validator_index(builder_index: BuilderIndex) -> ValidatorIndex:
//     return ValidatorIndex(builder_index | BUILDER_INDEX_FLAG)
func ConvertBuilderIndexToValidatorIndex(builderIndex primitives.BuilderIndex) primitives.ValidatorIndex {
	return primitives.ValidatorIndex(uint64(builderIndex) | params.BeaconConfig().BuilderIndexFlag)
}

// IsBuilderWithdrawalCredential returns true when the builder withdrawal prefix is set.
// Spec v1.6.1 (pseudocode):
// def is_builder_withdrawal_credential(withdrawal_credentials: Bytes32) -> bool:
//     return withdrawal_credentials[:1] == BUILDER_WITHDRAWAL_PREFIX
func IsBuilderWithdrawalCredential(withdrawalCredentials []byte) bool {
	if len(withdrawalCredentials) == 0 {
		return false
	}
	return withdrawalCredentials[0] == params.BeaconConfig().BuilderWithdrawalPrefixByte
}

// IsActiveBuilder checks if the builder is active for the given state.
// Spec v1.6.1 (pseudocode):
// def is_active_builder(state: BeaconState, builder_index: BuilderIndex) -> bool:
//     """
//     Check if the builder at ``builder_index`` is active for the given ``state``.
//     """
//     builder = state.builders[builder_index]
//     return (
//         # Placement in builder list is finalized
//         builder.deposit_epoch < state.finalized_checkpoint.epoch
//         # Has not initiated exit
//         and builder.withdrawable_epoch == FAR_FUTURE_EPOCH
//     )
func IsActiveBuilder(st state.ReadOnlyBeaconState, builderIndex primitives.BuilderIndex) bool {
	builders, err := st.Builders()
	if err != nil {
		return false
	}
	if builderIndex >= primitives.BuilderIndex(len(builders)) {
		return false
	}

	builder := builders[int(builderIndex)]
	if builder == nil {
		return false
	}

	finalizedEpoch := st.FinalizedCheckpointEpoch()
	return builder.DepositEpoch < finalizedEpoch &&
		builder.WithdrawableEpoch == params.BeaconConfig().FarFutureEpoch
}
