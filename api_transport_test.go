package papers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExpandedTransport(t *testing.T) {
	type params struct {
		Q        *string `query:"q,required"`
		Optional *bool   `query:"optional"`
	}
	called := 0
	c := New("fixture-token", WithDoer(func(_ *http.Client, r *http.Request) (*http.Response, error) {
		called++
		require.Equal(t, "false", r.URL.Query().Get("optional"))
		require.Contains(t, r.Header.Get("Accept"), "application/pdf")
		return response(200, "%PDF-synthetic"), nil
	}))
	_, err := callAPI[[]byte](c, context.Background(), "GET", "/document/telechargement", params{}, nil, responseBinary, []int{200})
	require.ErrorContains(t, err, "q")
	require.Zero(t, called)
	_, err = callAPI[[]byte](c, context.Background(), "POST", "/liste", params{Q: Pointer("fixture")}, make(chan int), responseBinary, []int{200})
	require.EqualError(t, err, "corps Papers JSON invalide")
	result, err := callAPI[[]byte](c, context.Background(), "GET", "/document/telechargement", params{Pointer("fixture"), Pointer(false)}, nil, responseBinary, []int{200})
	require.NoError(t, err)
	require.Equal(t, []byte("%PDF-synthetic"), result.Body)
	for _, raw := range []string{"null", "broken", `{"siren":42}`} {
		c := New("fixture-token", WithDoer(func(_ *http.Client, _ *http.Request) (*http.Response, error) { return response(200, raw), nil }))
		_, err := c.CompanyDetails(context.Background(), CompanyDetailsParams{})
		require.EqualError(t, err, "réponse Papers JSON invalide")
	}
	c = New("fixture-token", WithDoer(func(_ *http.Client, _ *http.Request) (*http.Response, error) { return response(200, ""), nil }))
	empty, err := callAPI[struct{}](c, context.Background(), "POST", "/liste-informations", struct{}{}, struct{}{}, responseEmpty, []int{200})
	require.NoError(t, err)
	require.Empty(t, empty.RawBody)
}

func TestExpandedTransportFailures(t *testing.T) {
	for _, raw := range []string{strings.Repeat("x", maxResponse+1)} {
		c := New("fixture-token", WithDoer(func(_ *http.Client, _ *http.Request) (*http.Response, error) { return response(200, raw), nil }))
		_, err := callAPI[[]byte](c, context.Background(), "GET", "/document/telechargement", struct{}{}, nil, responseBinary, []int{200})
		require.EqualError(t, err, "réponse Papers trop volumineuse")
	}
	c := New("fixture-token")
	_, _, _, err := c.request(context.Background(), "GET", "\n", nil, nil, false, []int{200})
	require.EqualError(t, err, "requête Papers invalide")
	c = New("fixture-token", WithDoer(func(_ *http.Client, _ *http.Request) (*http.Response, error) { return nil, errors.New("fixture-token") }))
	_, err = c.CompanyDetails(context.Background(), CompanyDetailsParams{})
	require.EqualError(t, err, "connexion Papers impossible")
}

func TestCompanyDetailsStatusAndOptions(t *testing.T) {
	c := New("fixture-token", WithDoer(func(_ *http.Client, r *http.Request) (*http.Response, error) {
		require.Equal(t, "001234567", r.URL.Query().Get("siren"))
		require.Equal(t, "false", r.URL.Query().Get("validite_tva_intracommunautaire"))
		return response(206, `{"siren":"001234567","statut_rcs":"Radié","statut_rne":"Radié","date_radiation_rcs":"2025-01-01","procedures_collectives":[{"type":"Liquidation judiciaire"}],"extension_future":{"valeur":1}}`), nil
	}))
	result, err := c.CompanyDetails(context.Background(), CompanyDetailsParams{Siren: Pointer("001234567"), ValiditeTvaIntracommunautaire: Pointer(false)})
	require.NoError(t, err)
	require.Equal(t, 206, result.StatusCode)
	require.Equal(t, "Radié", *result.Body.StatutRcs)
	require.Equal(t, "Radié", *result.Body.StatutRne)
	require.Equal(t, "2025-01-01", *result.Body.DateRadiationRcs)
	require.Len(t, result.Body.ProceduresCollectives, 1)
	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(result.RawBody, &raw))
	require.Contains(t, raw, "extension_future")
}
