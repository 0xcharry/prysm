package gloas

import (
	"bytes"
	"context"
	"fmt"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/electra"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/signing"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/state"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/interfaces"
	"github.com/OffchainLabs/prysm/v7/crypto/bls"
	"github.com/OffchainLabs/prysm/v7/encoding/ssz"
	enginev1 "github.com/OffchainLabs/prysm/v7/proto/engine/v1"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/time/slots"
	"github.com/pkg/errors"
)

// ProcessExecutionPayload processes the signed execution payload envelope for the Gloas fork.
// Spec v1.6.1 (pseudocode):
// def process_execution_payload(state, signed_envelope, execution_engine, verify=True):
//
//	envelope = signed_envelope.message
//	payload = envelope.payload
//
//	if verify: assert verify_execution_payload_envelope_signature(state, signed_envelope)
//
//	prev_root = hash_tree_root(state)
//	if state.latest_block_header.state_root == Root():
//	    state.latest_block_header.state_root = prev_root
//
//	assert envelope.beacon_block_root == hash_tree_root(state.latest_block_header)
//	assert envelope.slot == state.slot
//
//	committed_bid = state.latest_execution_payload_bid
//	assert envelope.builder_index == committed_bid.builder_index
//	assert committed_bid.blob_kzg_commitments_root == hash_tree_root(envelope.blob_kzg_commitments)
//	assert committed_bid.prev_randao == payload.prev_randao
//
//	assert hash_tree_root(payload.withdrawals) == hash_tree_root(state.payload_expected_withdrawals)
//
//	assert committed_bid.gas_limit == payload.gas_limit
//	assert committed_bid.block_hash == payload.block_hash
//	assert payload.parent_hash == state.latest_block_hash
//	assert payload.timestamp == compute_time_at_slot(state, state.slot)
//	assert (
//	    len(envelope.blob_kzg_commitments)
//	    <= get_blob_parameters(get_current_epoch(state)).max_blobs_per_block
//	)
//	versioned_hashes = [
//	    kzg_commitment_to_versioned_hash(commitment) for commitment in envelope.blob_kzg_commitments
//	]
//	requests = envelope.execution_requests
//	assert execution_engine.verify_and_notify_new_payload(
//	    NewPayloadRequest(
//	        execution_payload=payload,
//	        versioned_hashes=versioned_hashes,
//	        parent_beacon_block_root=state.latest_block_header.parent_root,
//	        execution_requests=requests,
//	    )
//	)
//
//	for op in requests.deposits: process_deposit_request(state, op)
//	for op in requests.withdrawals: process_withdrawal_request(state, op)
//	for op in requests.consolidations: process_consolidation_request(state, op)
//
//	payment = state.builder_pending_payments[SLOTS_PER_EPOCH + state.slot % SLOTS_PER_EPOCH]
//	amount = payment.withdrawal.amount
//	if amount > 0:
//	    exit_queue_epoch = compute_exit_epoch_and_update_churn(state, amount)
//	    payment.withdrawal.withdrawable_epoch = Epoch(exit_queue_epoch + MIN_VALIDATOR_WITHDRAWABILITY_DELAY)
//	    state.builder_pending_withdrawals.append(payment.withdrawal)
//	state.builder_pending_payments[SLOTS_PER_EPOCH + state.slot % SLOTS_PER_EPOCH] = BuilderPendingPayment()
//
//	state.execution_payload_availability[state.slot % SLOTS_PER_HISTORICAL_ROOT] = 0b1
//	state.latest_block_hash = payload.block_hash
//
//	if verify: assert envelope.state_root == h
func ProcessExecutionPayload(
	ctx context.Context,
	st state.BeaconState,
	signedEnvelope interfaces.ROSignedExecutionPayloadEnvelope,
) error {
	// Verify the signature on the signed execution payload envelope
	if err := verifyExecutionPayloadEnvelopeSignature(st, signedEnvelope); err != nil {
		return errors.Wrap(err, "signature verification failed")
	}

	envelope, err := signedEnvelope.Envelope()
	if err != nil {
		return errors.Wrap(err, "could not get envelope from signed envelope")
	}
	payload, err := envelope.Execution()
	if err != nil {
		return errors.Wrap(err, "could not get execution payload from envelope")
	}

	latestHeader := st.LatestBlockHeader()
	if len(latestHeader.StateRoot) == 0 || bytes.Equal(latestHeader.StateRoot, make([]byte, 32)) {
		previousStateRoot, err := st.HashTreeRoot(ctx)
		if err != nil {
			return errors.Wrap(err, "could not compute state root")
		}
		latestHeader.StateRoot = previousStateRoot[:]
		if err := st.SetLatestBlockHeader(latestHeader); err != nil {
			return errors.Wrap(err, "could not set latest block header")
		}
	}

	blockHeaderRoot, err := latestHeader.HashTreeRoot()
	if err != nil {
		return errors.Wrap(err, "could not compute block header root")
	}
	beaconBlockRoot := envelope.BeaconBlockRoot()
	if !bytes.Equal(beaconBlockRoot[:], blockHeaderRoot[:]) {
		return errors.Errorf("envelope beacon block root does not match state latest block header root: envelope=%#x, header=%#x", beaconBlockRoot, blockHeaderRoot)
	}

	if envelope.Slot() != st.Slot() {
		return errors.Errorf("envelope slot does not match state slot: envelope=%d, state=%d", envelope.Slot(), st.Slot())
	}

	latestBid, err := st.LatestExecutionPayloadBid()
	if err != nil {
		return errors.Wrap(err, "could not get latest execution payload bid")
	}
	if latestBid == nil {
		return errors.New("latest execution payload bid is nil")
	}
	if envelope.BuilderIndex() != latestBid.BuilderIndex() {
		return errors.Errorf("envelope builder index does not match committed bid builder index: envelope=%d, bid=%d", envelope.BuilderIndex(), latestBid.BuilderIndex())
	}

	envelopeBlobCommitments := envelope.BlobKzgCommitments()
	envelopeBlobRoot, err := ssz.KzgCommitmentsRoot(envelopeBlobCommitments)
	if err != nil {
		return errors.Wrap(err, "could not compute envelope blob KZG commitments root")
	}
	committedBlobRoot := latestBid.BlobKzgCommitmentsRoot()
	if !bytes.Equal(committedBlobRoot[:], envelopeBlobRoot[:]) {
		return errors.Errorf("committed bid blob KZG commitments root does not match envelope: bid=%#x, envelope=%#x", committedBlobRoot, envelopeBlobRoot)
	}

	withdrawals, err := payload.Withdrawals()
	if err != nil {
		return errors.Wrap(err, "could not get withdrawals from payload")
	}
	cfg := params.BeaconConfig()
	withdrawalsRoot, err := ssz.WithdrawalSliceRoot(withdrawals, cfg.MaxWithdrawalsPerPayload)
	if err != nil {
		return errors.Wrap(err, "could not compute withdrawals root")
	}

	latestWithdrawalsRoot, err := st.LatestWithdrawalsRoot()
	if err != nil {
		return errors.Wrap(err, "could not get latest withdrawals root")
	}
	if !bytes.Equal(withdrawalsRoot[:], latestWithdrawalsRoot[:]) {
		return errors.Errorf("payload withdrawals root does not match state latest withdrawals root: payload=%#x, state=%#x", withdrawalsRoot, latestWithdrawalsRoot)
	}

	if latestBid.GasLimit() != payload.GasLimit() {
		return errors.Errorf("committed bid gas limit does not match payload gas limit: bid=%d, payload=%d", latestBid.GasLimit(), payload.GasLimit())
	}

	latestBidPrevRandao := latestBid.PrevRandao()
	if !bytes.Equal(payload.PrevRandao(), latestBidPrevRandao[:]) {
		return errors.Errorf("payload prev randao does not match committed bid prev randao: payload=%#x, bid=%#x", payload.PrevRandao(), latestBidPrevRandao)
	}

	bidBlockHash := latestBid.BlockHash()
	payloadBlockHash := payload.BlockHash()
	if !bytes.Equal(bidBlockHash[:], payloadBlockHash) {
		return errors.Errorf("committed bid block hash does not match payload block hash: bid=%#x, payload=%#x", bidBlockHash, payloadBlockHash)
	}

	latestBlockHash, err := st.LatestBlockHash()
	if err != nil {
		return errors.Wrap(err, "could not get latest block hash")
	}
	if !bytes.Equal(payload.ParentHash(), latestBlockHash[:]) {
		return errors.Errorf("payload parent hash does not match state latest block hash: payload=%#x, state=%#x", payload.ParentHash(), latestBlockHash)
	}

	t, err := slots.StartTime(st.GenesisTime(), st.Slot())
	if err != nil {
		return errors.Wrap(err, "could not compute timestamp")
	}
	if payload.Timestamp() != uint64(t.Unix()) {
		return errors.Errorf("payload timestamp does not match expected timestamp: payload=%d, expected=%d", payload.Timestamp(), uint64(t.Unix()))
	}

	maxBlobsPerBlock := cfg.MaxBlobsPerBlock(envelope.Slot())
	if len(envelopeBlobCommitments) > maxBlobsPerBlock {
		return errors.Errorf("too many blob KZG commitments: got=%d, max=%d", len(envelopeBlobCommitments), maxBlobsPerBlock)
	}

	if err := processExecutionRequests(ctx, st, envelope.ExecutionRequests()); err != nil {
		return errors.Wrap(err, "could not process execution requests")
	}

	if err := queueBuilderPayment(ctx, st); err != nil {
		return errors.Wrap(err, "could not queue builder payment")
	}

	if err := st.SetExecutionPayloadAvailability(st.Slot(), true); err != nil {
		return errors.Wrap(err, "could not set execution payload availability")
	}

	if err := st.SetLatestBlockHash([32]byte(payload.BlockHash())); err != nil {
		return errors.Wrap(err, "could not set latest block hash")
	}

	r, err := st.HashTreeRoot(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get hash tree root")
	}
	if r != envelope.StateRoot() {
		return fmt.Errorf("state root mismatch: expected %#x, got %#x", envelope.StateRoot(), r)
	}

	return nil
}

// processExecutionRequests processes deposits, withdrawals, and consolidations from execution requests.
// Spec v1.6.1 (pseudocode):
// for op in requests.deposits: process_deposit_request(state, op)
// for op in requests.withdrawals: process_withdrawal_request(state, op)
// for op in requests.consolidations: process_consolidation_request(state, op)
func processExecutionRequests(ctx context.Context, st state.BeaconState, requests *enginev1.ExecutionRequests) error {
	var err error
	st, err = electra.ProcessDepositRequests(ctx, st, requests.Deposits)
	if err != nil {
		return errors.Wrap(err, "could not process deposit requests")
	}

	st, err = electra.ProcessWithdrawalRequests(ctx, st, requests.Withdrawals)
	if err != nil {
		return errors.Wrap(err, "could not process withdrawal requests")
	}
	err = electra.ProcessConsolidationRequests(ctx, st, requests.Consolidations)
	if err != nil {
		return errors.Wrap(err, "could not process consolidation requests")
	}
	return nil
}

// queueBuilderPayment implements the builder payment queuing logic for Gloas.
// Spec v1.6.1 (pseudocode):
// payment = state.builder_pending_payments[SLOTS_PER_EPOCH + state.slot % SLOTS_PER_EPOCH]
// amount = payment.withdrawal.amount
// if amount > 0:
//
//	exit_queue_epoch = compute_exit_epoch_and_update_churn(state, amount)
//	payment.withdrawal.withdrawable_epoch = Epoch(exit_queue_epoch + MIN_VALIDATOR_WITHDRAWABILITY_DELAY)
//	state.builder_pending_withdrawals.append(payment.withdrawal)
//
// state.builder_pending_payments[SLOTS_PER_EPOCH + state.slot % SLOTS_PER_EPOCH] = BuilderPendingPayment()
func queueBuilderPayment(ctx context.Context, st state.BeaconState) error {
	payment, err := st.BuilderPendingPayment(st.Slot())
	if err != nil {
		return errors.Wrap(err, "could not get builder pending payment")
	}

	amount := payment.Withdrawal.Amount
	if amount > 0 {
		exitQueueEpoch, err := st.ExitEpochAndUpdateChurn(amount)
		if err != nil {
			return errors.Wrap(err, "could not compute exit epoch and update churn")
		}

		minValidatorWithdrawabilityDelay := params.BeaconConfig().MinValidatorWithdrawabilityDelay
		payment.Withdrawal.WithdrawableEpoch = exitQueueEpoch + minValidatorWithdrawabilityDelay

		if err := st.AppendBuilderPendingWithdrawal(payment.Withdrawal); err != nil {
			return errors.Wrap(err, "could not append builder pending withdrawal")
		}
	}

	emptyPayment := &ethpb.BuilderPendingPayment{
		Withdrawal: &ethpb.BuilderPendingWithdrawal{
			FeeRecipient: make([]byte, 20),
		},
	}
	if err := st.SetBuilderPendingPayment(st.Slot(), emptyPayment); err != nil {
		return errors.Wrap(err, "could not set builder pending payment")
	}

	return nil
}

// verifyExecutionPayloadEnvelopeSignature verifies the BLS signature on a signed execution payload envelope.
// Spec v1.6.1 (pseudocode):
// if verify: assert verify_execution_payload_envelope_signature(state, signed_envelope)
func verifyExecutionPayloadEnvelopeSignature(st state.BeaconState, signedEnvelope interfaces.ROSignedExecutionPayloadEnvelope) error {
	envelope, err := signedEnvelope.Envelope()
	if err != nil {
		return fmt.Errorf("failed to get envelope: %w", err)
	}

	builderPubkey := st.PubkeyAtIndex(envelope.BuilderIndex())
	publicKey, err := bls.PublicKeyFromBytes(builderPubkey[:])
	if err != nil {
		return fmt.Errorf("invalid builder public key: %w", err)
	}

	signatureBytes := signedEnvelope.Signature()
	signature, err := bls.SignatureFromBytes(signatureBytes[:])
	if err != nil {
		return fmt.Errorf("invalid signature format: %w", err)
	}

	currentEpoch := slots.ToEpoch(envelope.Slot())
	domain, err := signing.Domain(
		st.Fork(),
		currentEpoch,
		params.BeaconConfig().DomainBeaconBuilder,
		st.GenesisValidatorsRoot(),
	)
	if err != nil {
		return fmt.Errorf("failed to compute signing domain: %w", err)
	}

	signingRoot, err := signedEnvelope.SigningRoot(domain)
	if err != nil {
		return fmt.Errorf("failed to compute signing root: %w", err)
	}

	if !signature.Verify(publicKey, signingRoot[:]) {
		return fmt.Errorf("signature verification failed: %w", signing.ErrSigFailedToVerify)
	}

	return nil
}
