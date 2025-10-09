package state_native

import (
	"testing"

	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/stretchr/testify/require"
)

func TestUpdateExecutionPayloadAvailabilityAtIndex_SetAndClear(t *testing.T) {
	st := newGloasStateWithAvailability(t, make([]byte, 1024))

	otherIdx := uint64(8) // byte 1, bit 0
	idx := uint64(9)      // byte 1, bit 1

	require.NoError(t, st.UpdateExecutionPayloadAvailabilityAtIndex(otherIdx, 1))
	require.Equal(t, byte(0x01), st.executionPayloadAvailability[1])

	require.NoError(t, st.UpdateExecutionPayloadAvailabilityAtIndex(idx, 1))
	require.Equal(t, byte(0x03), st.executionPayloadAvailability[1])

	require.NoError(t, st.UpdateExecutionPayloadAvailabilityAtIndex(idx, 0))
	require.Equal(t, byte(0x01), st.executionPayloadAvailability[1])
}

func TestUpdateExecutionPayloadAvailabilityAtIndex_OutOfRange(t *testing.T) {
	st := newGloasStateWithAvailability(t, make([]byte, 1024))

	idx := uint64(len(st.executionPayloadAvailability)) * 8
	err := st.UpdateExecutionPayloadAvailabilityAtIndex(idx, 1)
	require.Error(t, err)
	require.ErrorContains(t, err, "out of range")

	for _, b := range st.executionPayloadAvailability {
		if b != 0 {
			t.Fatalf("execution payload availability mutated on error")
		}
	}
}

func newGloasStateWithAvailability(t *testing.T, availability []byte) *BeaconState {
	t.Helper()

	st, err := InitializeFromProtoUnsafeGloas(&ethpb.BeaconStateGloas{
		ExecutionPayloadAvailability: availability,
	})
	require.NoError(t, err)

	return st.(*BeaconState)
}
