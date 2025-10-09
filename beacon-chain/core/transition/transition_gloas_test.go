package transition

import (
	"context"
	"errors"
	"testing"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/state"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/runtime/version"
	"github.com/stretchr/testify/require"
)

type trackingState struct {
	state.BeaconState
	slot       primitives.Slot
	version    int
	header     *ethpb.BeaconBlockHeader
	epaCalls   int
	epaIndex   uint64
	epaValue   byte
	epaErr     error
}

func (s *trackingState) Slot() primitives.Slot {
	return s.slot
}

func (s *trackingState) Version() int {
	return s.version
}

func (s *trackingState) HashTreeRoot(_ context.Context) ([32]byte, error) {
	return [32]byte{0x01}, nil
}

func (s *trackingState) UpdateStateRootAtIndex(_ uint64, _ [32]byte) error {
	return nil
}

func (s *trackingState) LatestBlockHeader() *ethpb.BeaconBlockHeader {
	return s.header
}

func (s *trackingState) SetLatestBlockHeader(header *ethpb.BeaconBlockHeader) error {
	s.header = header
	return nil
}

func (s *trackingState) UpdateBlockRootAtIndex(_ uint64, _ [32]byte) error {
	return nil
}

func (s *trackingState) UpdateExecutionPayloadAvailabilityAtIndex(idx uint64, val byte) error {
	s.epaCalls++
	s.epaIndex = idx
	s.epaValue = val
	if s.epaErr != nil {
		return s.epaErr
	}
	return nil
}

func TestProcessSlot_GloasClearsNextPayloadAvailability(t *testing.T) {
	st := &trackingState{
		slot:    10,
		version: version.Gloas,
		header:  testBeaconBlockHeader(),
	}

	_, err := ProcessSlot(context.Background(), st)
	require.NoError(t, err)
	require.Equal(t, 1, st.epaCalls)
	require.Equal(t, uint64((st.slot+1)%params.BeaconConfig().SlotsPerHistoricalRoot), st.epaIndex)
	require.Equal(t, byte(0x0), st.epaValue)
}

func TestProcessSlot_GloasAvailabilityUpdateError(t *testing.T) {
	updateErr := errors.New("update failed")
	st := &trackingState{
		slot:    7,
		version: version.Gloas,
		header:  testBeaconBlockHeader(),
		epaErr:  updateErr,
	}

	_, err := ProcessSlot(context.Background(), st)
	require.ErrorIs(t, err, updateErr)
	require.Equal(t, 1, st.epaCalls)
}

func testBeaconBlockHeader() *ethpb.BeaconBlockHeader {
	return &ethpb.BeaconBlockHeader{
		ParentRoot: make([]byte, 32),
		StateRoot:  make([]byte, 32),
		BodyRoot:   make([]byte, 32),
	}
}
