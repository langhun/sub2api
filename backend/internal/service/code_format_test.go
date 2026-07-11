package service

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCodeFormatConfigGenerate(t *testing.T) {
	config := CodeFormatConfig{
		Prefix:       "BAL",
		CharacterSet: CodeCharacterSetHex,
		Separator:    "-",
		GroupLength:  4,
		GroupCount:   3,
	}

	code, err := config.generate(bytes.NewReader(bytes.Repeat([]byte{0, 1, 2, 3}, 32)))
	require.NoError(t, err)
	require.Equal(t, "BAL-0123-1230-2301", code)
}

func TestCodeFormatConfigGenerateWithoutSeparator(t *testing.T) {
	config := CodeFormatConfig{
		Prefix:       "RP",
		CharacterSet: CodeCharacterSetNumeric,
		GroupLength:  4,
		GroupCount:   1,
	}

	code, err := config.generate(bytes.NewReader(bytes.Repeat([]byte{1, 2, 3, 4}, 8)))
	require.NoError(t, err)
	require.Equal(t, "RP1234", code)
}

func TestCodeFormatConfigDefaultsPreserveExistingShapes(t *testing.T) {
	redeem := DefaultRedeemCodeFormat()
	redeemCode, err := redeem.Generate()
	require.NoError(t, err)
	require.Len(t, redeemCode, 35)
	require.Len(t, strings.Split(redeemCode, "-"), 4)

	compact := DefaultCompactRedeemCodeFormat()
	compactCode, err := compact.Generate()
	require.NoError(t, err)
	require.Len(t, compactCode, 32)
	require.NotContains(t, compactCode, "-")

	redPacket := DefaultRedPacketCodeFormat()
	redPacketCode, err := redPacket.Generate()
	require.NoError(t, err)
	require.Len(t, redPacketCode, 24)
}

func TestCodeFormatConfigValidation(t *testing.T) {
	tests := []CodeFormatConfig{
		{CharacterSet: "unknown", GroupLength: 4, GroupCount: 1},
		{CharacterSet: CodeCharacterSetHex, GroupLength: 0, GroupCount: 1},
		{CharacterSet: CodeCharacterSetHex, GroupLength: 4, GroupCount: 0},
		{Prefix: "BAD PREFIX", CharacterSet: CodeCharacterSetHex, GroupLength: 4, GroupCount: 1},
		{Separator: "--", CharacterSet: CodeCharacterSetHex, GroupLength: 4, GroupCount: 1},
		{Prefix: "A-B", Separator: "-", CharacterSet: CodeCharacterSetHex, GroupLength: 4, GroupCount: 1},
	}

	for _, config := range tests {
		require.Error(t, config.Validate())
	}
}

func TestCodeFormatConfigPropagatesRandomFailure(t *testing.T) {
	config := DefaultRedeemCodeFormat()
	_, err := config.generate(errorReader{})
	require.Error(t, err)
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) {
	return 0, errors.New("random source unavailable")
}
