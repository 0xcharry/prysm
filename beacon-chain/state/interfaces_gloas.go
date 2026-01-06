package state

import (
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	enginev1 "github.com/OffchainLabs/prysm/v7/proto/engine/v1"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
)

// Builder represents a builder registry entry.
type Builder struct {
	ExecutionAddress []byte
	Balance          uint64
	DepositEpoch     primitives.Epoch
	WithdrawableEpoch primitives.Epoch
}

// ReadOnlyGloas defines read access to Gloas-specific state fields.
type ReadOnlyGloas interface {
	IsParentBlockFull() bool
	BuildersCount() int
	Builders() ([]*Builder, error)
	NextWithdrawalBuilderIndex() (primitives.BuilderIndex, error)
	BuilderPendingWithdrawals() ([]*ethpb.BuilderPendingWithdrawal, error)
}

// WriteOnlyGloas defines write access to Gloas-specific state fields.
type WriteOnlyGloas interface {
	SetPayloadExpectedWithdrawals(withdrawals []*enginev1.Withdrawal) error
	DequeueBuilderPendingWithdrawals(num uint64) error
	SetNextWithdrawalBuilderIndex(idx primitives.BuilderIndex) error
}
