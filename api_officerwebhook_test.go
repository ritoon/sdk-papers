package papers

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestOfficerWebhook(t *testing.T) {
	payload := `{"evenement":"notifications.dirigeants","date_envoi":"2025-01-01 00:00:00","donnees":{"id_list":"synthetic-list","nom_list":"Fixture","dirigeants":[{"siren":"001234567"}]}}`
	result, err := DecodeOfficerWebhook([]byte(payload))
	require.NoError(t, err)
	require.Equal(t, "notifications.dirigeants", *result.Evenement)
	require.Equal(t, "synthetic-list", *result.Donnees.IdList)
	require.Len(t, result.Donnees.Dirigeants, 1)
	for _, invalid := range []string{"null", "[]", "broken", "{}", `{"evenement":"wrong","donnees":{}}`, `{"evenement":"notifications.dirigeants","donnees":null}`, `{"evenement":"notifications.dirigeants","donnees":[]}`, `{"evenement":"notifications.dirigeants"}`, strings.Repeat("x", maxResponse+1)} {
		result, err := DecodeOfficerWebhook([]byte(invalid))
		require.Error(t, err)
		require.Nil(t, result)
	}
}
