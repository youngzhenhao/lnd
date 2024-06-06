package lnwallet

import (
	"github.com/btcsuite/btcd/wire"
	"github.com/lightningnetwork/lnd/fn"
	"github.com/lightningnetwork/lnd/input"
	"github.com/lightningnetwork/lnd/tlv"
)

// ResolutionReq..
type ResolutionReq struct {
	// ChanPoint...
	ChanPoint wire.OutPoint

	// CommitBlob...
	CommitBlob fn.Option[tlv.Blob]

	// Type...
	Type input.WitnessType

	// CommitTx...
	CommitTx *wire.MsgTx

	// ContractPoint...
	ContractPoint wire.OutPoint

	// SignDesc...
	SignDesc input.SignDescriptor

	// KeyRing...
	KeyRing *CommitmentKeyRing

	// CsvDelay...
	CsvDelay fn.Option[uint32]

	// CltvDelay...
	CltvDelay fn.Option[uint32]
}

// AuxContractResolver...
type AuxContractResolver interface {
	// ResolveContract...
	//
	// * cisc or risc?
	// * for each of given method, etc?
	ResolveContract(ResolutionReq) fn.Result[tlv.Blob]
}
