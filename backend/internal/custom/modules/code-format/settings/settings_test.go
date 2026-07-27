package settings

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValuesRoundTripAndPreservesLegacyHexCompatibility(t *testing.T) {
	config := Default()
	config.Balance = Format{Prefix: "BAL", CharacterSet: CharacterSetNumeric, Separator: "-", GroupLength: 2, GroupCount: 2}
	values, err := Values(config)
	require.NoError(t, err)
	require.Equal(t, config, FromValues(values))

	legacy := FromValues(map[string]string{KeyInvitation: `{"prefix":"INV","character_set":"hex","separator":"","group_length":3,"group_count":1}`})
	require.Equal(t, CharacterSetAlphanumeric, legacy.Invitation.CharacterSet)
	require.Equal(t, "INV", legacy.Invitation.Prefix)
}

func TestReaderUsesExistingKeysWithoutWritingDefaults(t *testing.T) {
	store := &settingsStore{values: map[string]string{KeyRedPacket: `{"prefix":"RP","character_set":"numeric","separator":"","group_length":4,"group_count":1}`}}
	config, err := New(store).Read(context.Background())
	require.NoError(t, err)
	require.Equal(t, "RP", config.RedPacket.Prefix)
	require.Zero(t, store.writes)
}

type settingsStore struct {
	values map[string]string
	writes int
}

func (s *settingsStore) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}

func (s *settingsStore) SetMultiple(_ context.Context, values map[string]string) error {
	s.writes++
	for key, value := range values {
		s.values[key] = value
	}
	return nil
}
