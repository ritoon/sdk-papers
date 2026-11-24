package papers

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestCompanyWebhook(t *testing.T) {
	payload := `{"evenement":"notifications.entreprises","date_envoi":"2025-01-01 00:00:00","donnees":{"id_list":"synthetic-list","nom_list":"Fixture","entreprises":[{"siren":"001234567"}]}}`
	result, err := DecodeCompanyWebhook([]byte(payload))
	require.NoError(t, err)
	require.Equal(t, "notifications.entreprises", *result.Evenement)
	require.Equal(t, "synthetic-list", *result.Donnees.IdList)
	require.Len(t, result.Donnees.Entreprises, 1)
	for _, invalid := range []string{"null", "[]", "broken", "{}", `{"evenement":"wrong","donnees":{}}`, `{"evenement":"notifications.entreprises","donnees":null}`, `{"evenement":"notifications.entreprises","donnees":[]}`, `{"evenement":"notifications.entreprises"}`, strings.Repeat("x", maxResponse+1)} {
		result, err := DecodeCompanyWebhook([]byte(invalid))
		require.Error(t, err)
		require.Nil(t, result)
	}
}
