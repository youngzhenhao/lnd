package lnwallet

import (
	"github.com/btcsuite/btcd/wire"
	"github.com/lightningnetwork/lnd/fn"
	"github.com/lightningnetwork/lnd/tlv"
)

// ContractType...
type ContractType uint8

const (
	// CommitNoDelay...
	CommitNoDelay = iota

	// CommitDelay...
	CommitDelay = iota
)

// ResolutionReq..
type ResolutionReq struct {
	// ChanPoint...
	ChanPoint wire.OutPoint

	// CommitBlob...
	CommitBlob fn.Option[tlv.Blob]

	// Type...
	Type ContractType
}

// AuxContractResolver...
type AuxContractResolver interface {
	// ResolveContract...
	//
	// * cisc or risc?
	// * for each of given method, etc?
	ResolveContract(ResolutionReq) fn.Result[tlv.Blob]
}
