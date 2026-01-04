package state

import (
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
)

type WriteOnlyGloasFields interface {
	SetBuilderPendingPayment(index int, payment *ethpb.BuilderPendingPayment) error
}
