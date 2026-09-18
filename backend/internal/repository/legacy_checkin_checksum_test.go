package repository

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

func TestLegacyCheckinDateRepairChecksumCompatibility(t *testing.T) {
	for name, previous := range map[string]string{
		"177_migrate_legacy_checkin_records.sql":  "e2418fb16e4ba277ff48ca03f2da78255642671d4edbf8d5cf69a851d8ad7c9c",
		"179_repair_legacy_checkin_migration.sql": "92d5a648bbe370122f6e486e123c5a13e49e7f68feb61171b70fb2ddab9df467",
	} {
		t.Run(name, func(t *testing.T) {
			content, err := migrations.FS.ReadFile(name)
			require.NoError(t, err)
			current := fmt.Sprintf("%x", sha256.Sum256([]byte(strings.TrimSpace(string(content)))))
			require.True(t, isMigrationChecksumCompatible(name, previous, current))
			require.True(t, isMigrationChecksumCompatible(name, current, previous))
			require.False(t, isMigrationChecksumCompatible(name, "unknown", current))
			require.False(t, isMigrationChecksumCompatible(name, previous, "unknown"))
		})
	}
}
