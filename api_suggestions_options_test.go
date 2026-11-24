package papers

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestSuggestionsAllTargetsAndNumericFlags(t *testing.T) {
	for _, value := range []string{"true", "false", "0", "1", "null", "2", `"0"`} {
		c := New("fixture-token", WithDoer(func(_ *http.Client, req *http.Request) (*http.Response, error) {
			require.Equal(t, "a", req.URL.Query().Get("q"))
			require.Equal(t, "denomination,nom_complet,representant", req.URL.Query().Get("cibles"))
			return response(200, `{"resultats_denomination":[{"siren":"001234567","entreprise_cessee":`+value+`}],"resultats_nom_complet":[],"resultats_representant":[]}`), nil
		}))
		result, err := c.SuggestionsWithOptions(context.Background(), SuggestionsWithOptionsParams{Q: Pointer("a"), Cibles: Pointer("denomination,nom_complet,representant")})
		if value == "2" || value == `"0"` {
			require.Error(t, err)
			continue
		}
		require.NoError(t, err)
		flag := result.Body.ResultatsDenomination[0].EntrepriseCessee
		if value == "null" {
			require.Nil(t, flag)
			continue
		}
		require.Equal(t, value == "1" || value == "true", bool(*flag))
		raw, err := json.Marshal(flag)
		require.NoError(t, err)
		require.Contains(t, []string{"true", "false"}, string(raw))
	}
}
