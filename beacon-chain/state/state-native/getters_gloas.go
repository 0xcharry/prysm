package state_native

import (
	"bytes"

	state "github.com/OffchainLabs/prysm/v7/beacon-chain/state"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/runtime/version"
)

// IsParentBlockFull returns true when the latest bid was fulfilled with a payload.
func (b *BeaconState) IsParentBlockFull() bool {
	if b.version < version.Gloas {
		return false
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.latestExecutionPayloadBid == nil {
		return false
	}

	return bytes.Equal(b.latestExecutionPayloadBid.BlockHash, b.latestBlockHash)
}

// BuilderPendingWithdrawals returns the builder pending withdrawals queue.
func (b *BeaconState) BuilderPendingWithdrawals() ([]*ethpb.BuilderPendingWithdrawal, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.builderPendingWithdrawalsVal(), nil
}

// BuildersCount returns the number of builders in the registry.
func (b *BeaconState) BuildersCount() int {
	return 0
}

// Builders returns the builders registry.
func (b *BeaconState) Builders() ([]*state.Builder, error) {
	if b.version < version.Gloas {
		return nil, errNotSupported("Builders", b.version)
	}

	return nil, nil
}

// NextWithdrawalBuilderIndex returns the next builder index for the withdrawals sweep.
func (b *BeaconState) NextWithdrawalBuilderIndex() (primitives.BuilderIndex, error) {
	if b.version < version.Gloas {
		return 0, errNotSupported("NextWithdrawalBuilderIndex", b.version)
	}

	return 0, nil
}
