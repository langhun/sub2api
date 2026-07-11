package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRedPacketExpiryServiceLifecycleWithoutTransfer(t *testing.T) {
	service := NewRedPacketExpiryService(nil, time.Millisecond)
	require.NotPanics(t, service.Start)
	require.NotPanics(t, service.Stop)
	require.NotPanics(t, service.Stop)
}
