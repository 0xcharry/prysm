package state

import (
	"github.com/OffchainLabs/prysm/v7/consensus-types/interfaces"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
)

type WriteOnlyGloasFields interface {
	SetExecutionPayloadBid(h interfaces.ROExecutionPayloadBid) error
	SetBuilderPendingPayment(index primitives.Slot, payment *ethpb.BuilderPendingPayment) error
	SetLatestBlockHash(hash [32]byte) error
	SetExecutionPayloadAvailability(index primitives.Slot, available bool) error
	SetBuilderPendingPayments(payments []*ethpb.BuilderPendingPayment) error
	AppendBuilderPendingWithdrawal(withdrawal *ethpb.BuilderPendingWithdrawal) error
}

type ReadOnlyGloasFields interface {
	LatestBlockHash() ([32]byte, error)
	BuilderPendingPayment(slot primitives.Slot) (*ethpb.BuilderPendingPayment, error)
	BuilderPendingPayments() ([]*ethpb.BuilderPendingPayment, error)
	BuilderPendingWithdrawals() ([]*ethpb.BuilderPendingWithdrawal, error)
	LatestExecutionPayloadBid() (interfaces.ROExecutionPayloadBid, error)
	LatestWithdrawalsRoot() ([32]byte, error)
}
