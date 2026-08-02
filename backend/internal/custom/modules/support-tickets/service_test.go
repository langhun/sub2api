package supporttickets

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateCreateRejectsInvalidCategoryAndControlCharacters(t *testing.T) {
	require.Error(t, validateCreate(CreateInput{UserID: 1, Category: "unknown", Subject: "Need help", Body: "hello"}))
	require.NoError(t, validateCreate(CreateInput{UserID: 1, Category: "api", Subject: "Need help", Body: "password reset does not work"}))
	require.NoError(t, validateCreate(CreateInput{UserID: 1, Category: "api", Subject: "Need help", Body: "request id: abc"}))
	require.Error(t, validateCreate(CreateInput{UserID: 1, Category: "api", Subject: "Need help", Body: "line\rbreak"}))
}

func TestPublicIDShape(t *testing.T) {
	publicID, err := newPublicID()
	require.NoError(t, err)
	require.True(t, validPublicID(publicID))
	require.True(t, strings.HasPrefix(publicID, "SUP-"))
}
