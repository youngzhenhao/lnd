package record

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/lightningnetwork/lnd/lnwire"
	"github.com/stretchr/testify/require"
)

//nolint:lll
const pubkeyStr = "02eec7245d6b7d2ccb30380bfbe2a3648cd7a942653f5aa340edcea1f283686619"

func pubkey(t *testing.T) *btcec.PublicKey {
	t.Helper()

	nodeBytes, err := hex.DecodeString(pubkeyStr)
	require.NoError(t, err)

	nodePk, err := btcec.ParsePubKey(nodeBytes)
	require.NoError(t, err)

	return nodePk
}

// TestBlindedDataEncoding tests encoding and decoding of blinded data blobs.
// These tests specifically cover cases where the variable length encoded
// integers values have different numbers of leading zeros trimmed because
// these TLVs are the first composite records with variable length tlvs
// (previously, a variable length integer would take up the whole record).
func TestBlindedDataEncoding(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		baseFee     uint32
		htlcMin     lnwire.MilliSatoshi
		features    *lnwire.FeatureVector
		constraints bool
	}{
		{
			name:    "zero variable values",
			baseFee: 0,
			htlcMin: 0,
		},
		{
			name:    "zeros trimmed",
			baseFee: math.MaxUint32 / 2,
			htlcMin: math.MaxUint64 / 2,
		},
		{
			name:    "no zeros trimmed",
			baseFee: math.MaxUint32,
			htlcMin: math.MaxUint64,
		},
		{
			name:     "nil feature vector",
			features: nil,
		},
		{
			name:     "non-nil, but empty feature vector",
			features: lnwire.EmptyFeatureVector(),
		},
		{
			name: "populated feature vector",
			features: lnwire.NewFeatureVector(
				lnwire.NewRawFeatureVector(lnwire.AMPOptional),
				lnwire.Features,
			),
		},
		{
			name:        "no payment constraints",
			constraints: true,
		},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// Create a standard set of blinded route data, using
			// the values from our test case for the variable
			// length encoded values.
			channelID := lnwire.NewShortChanIDFromInt(1)
			info := PaymentRelayInfo{
				FeeRate:         2,
				CltvExpiryDelta: 3,
				BaseFee:         testCase.baseFee,
			}

			var constraints *PaymentConstraints
			if testCase.constraints {
				constraints = &PaymentConstraints{
					MaxCltvExpiry:   4,
					HtlcMinimumMsat: testCase.htlcMin,
				}
			}

			encodedData := NewBlindedRouteData(
				channelID, pubkey(t), info, constraints,
				testCase.features,
			)

			encoded, err := EncodeBlindedRouteData(encodedData)
			require.NoError(t, err)

			b := bytes.NewBuffer(encoded)
			decodedData, err := DecodeBlindedRouteData(b)
			require.NoError(t, err)

			require.Equal(t, encodedData, decodedData)
		})
	}
}

// TestBlindedRouteVectors tests encoding/decoding of the test vectors for
// blinded route data provided in the specification.
//
//nolint:lll
func TestBlindingSpecTestVectors(t *testing.T) {
	nextBlindingOverrideStr, err := hex.DecodeString("031b84c5567b126440995d3ed5aaba0565d71e1834604819ff9c17f5e9d5dd078f")
	require.NoError(t, err)
	nextBlindingOverride, err := btcec.ParsePubKey(nextBlindingOverrideStr)
	require.NoError(t, err)

	tests := []struct {
		encoded             string
		expectedPaymentData *BlindedRouteData
	}{
		{
			encoded: "011a0000000000000000000000000000000000000000000000000000020800000000000006c10a0800240000009627100c06000b69e505dc0e00fd023103123456",
			expectedPaymentData: NewBlindedRouteData(
				lnwire.ShortChannelID{
					BlockHeight: 0,
					TxIndex:     0,
					TxPosition:  1729,
				},
				nil,
				PaymentRelayInfo{
					CltvExpiryDelta: 36,
					FeeRate:         150,
					BaseFee:         10000,
				},
				&PaymentConstraints{
					MaxCltvExpiry:   748005,
					HtlcMinimumMsat: 1500,
				},
				lnwire.NewFeatureVector(
					lnwire.NewRawFeatureVector(),
					lnwire.Features,
				),
			),
		},
		{
			encoded: "020800000000000004510821031b84c5567b126440995d3ed5aaba0565d71e1834604819ff9c17f5e9d5dd078f0a0800300000006401f40c06000b69c105dc0e00",
			expectedPaymentData: NewBlindedRouteData(
				lnwire.ShortChannelID{
					TxPosition: 1105,
				},
				nextBlindingOverride,
				PaymentRelayInfo{
					CltvExpiryDelta: 48,
					FeeRate:         100,
					BaseFee:         500,
				},
				&PaymentConstraints{
					MaxCltvExpiry:   747969,
					HtlcMinimumMsat: 1500,
				},
				lnwire.NewFeatureVector(
					lnwire.NewRawFeatureVector(),
					lnwire.Features,
				)),
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			route, err := hex.DecodeString(test.encoded)
			require.NoError(t, err)

			buff := bytes.NewBuffer(route)

			decodedRoute, err := DecodeBlindedRouteData(buff)
			require.NoError(t, err)

			require.Equal(
				t, test.expectedPaymentData, decodedRoute,
			)
		})
	}
}
