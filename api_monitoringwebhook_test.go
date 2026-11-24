package papers

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestMonitoringWebhook(t *testing.T) {
	payload := `{"evenement":"notifications.veille","date_envoi":"2025-01-01 00:00:00","donnees":{"id_list":"synthetic-list","nom_list":"Fixture","siren_entrant":[{"siren":"001234567"}]}}`
	result, err := DecodeMonitoringWebhook([]byte(payload))
	require.NoError(t, err)
	require.Equal(t, "notifications.veille", *result.Evenement)
	require.Equal(t, "synthetic-list", *result.Donnees.IdList)
	require.Len(t, result.Donnees.SirenEntrant, 1)
	for _, invalid := range []string{"null", "[]", "broken", "{}", `{"evenement":"wrong","donnees":{}}`, `{"evenement":"notifications.veille","donnees":null}`, `{"evenement":"notifications.veille","donnees":[]}`, `{"evenement":"notifications.veille"}`, strings.Repeat("x", maxResponse+1)} {
		result, err := DecodeMonitoringWebhook([]byte(invalid))
		require.Error(t, err)
		require.Nil(t, result)
	}
}
