package codeformat

import (
	"context"
	"testing"

	codeformatsettings "github.com/Wei-Shaw/sub2api/internal/custom/modules/code-format/settings"
	customsettings "github.com/Wei-Shaw/sub2api/internal/custom/settings"
	"github.com/stretchr/testify/require"
)

func TestGeneratorReadsExistingFormatKeysWithoutWriting(t *testing.T) {
	store := &generatorStore{values: map[string]string{
		codeformatsettings.KeyBalance:   `{"prefix":"BAL","character_set":"numeric","separator":"-","group_length":2,"group_count":2}`,
		codeformatsettings.KeyRedPacket: `{"prefix":"RP","character_set":"numeric","separator":"","group_length":3,"group_count":1}`,
	}}
	generator := NewGenerator(customsettings.NewRegistry(store))

	code, err := generator.GenerateCode(context.Background(), "balance")
	require.NoError(t, err)
	require.Regexp(t, `^BAL-[0-9]{2}-[0-9]{2}$`, code)

	redPacket, err := generator.GenerateRedPacketCode(context.Background())
	require.NoError(t, err)
	require.Regexp(t, `^RP[0-9]{3}$`, redPacket)
	require.Zero(t, store.writes)
}

type generatorStore struct {
	values map[string]string
	writes int
}

func (s *generatorStore) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}

func (s *generatorStore) SetMultiple(_ context.Context, values map[string]string) error {
	s.writes++
	for key, value := range values {
		s.values[key] = value
	}
	return nil
}
