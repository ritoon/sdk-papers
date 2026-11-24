package papers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type transport func(*http.Request) (*http.Response, error)

func (f transport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func response(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}
}

func TestReadOperations(t *testing.T) {
	calls := 0
	c := New(" secret-token ", WithHTTPClient(&http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, "https", r.URL.Scheme)
		require.Equal(t, "api.pappers.fr", r.URL.Host)
		require.Equal(t, "GET", r.Method)
		require.Equal(t, "secret-token", r.Header.Get("api-key"))
		require.NotContains(t, r.URL.String(), "secret-token")
		deadline, ok := r.Context().Deadline()
		require.True(t, ok)
		require.LessOrEqual(t, time.Until(deadline), 25*time.Second)
		switch r.URL.Path {
		case "/v2/suggestions":
			require.Equal(t, "École & associés", r.URL.Query().Get("q"))
			require.Equal(t, "5", r.URL.Query().Get("longueur"))
			require.Equal(t, "nom_entreprise,siren,siret", r.URL.Query().Get("cibles"))
			return response(200, `{"resultats_nom_entreprise":[{"siren":"001234567","nom_entreprise":null,"siege":null,"diffusable":false}],"resultats_siren":[],"resultats_siret":[]}`), nil
		case "/v2/entreprise":
			require.Len(t, r.URL.Query(), 1)
			if r.URL.Query().Get("siret") != "" {
				require.Equal(t, "00123456700012", r.URL.Query().Get("siret"))
			} else {
				require.Equal(t, "001234567", r.URL.Query().Get("siren"))
			}
			return response(200, `{"siren":"001234567","numero_tva_intracommunautaire":"FR12001234567","etablissement":{"siret":"00123456700012"},"siege":{"siret":"00123456700012","adresse_ligne_1":"1 rue Test","code_postal":"75001","ville":"Paris"}}`), nil
		case "/v2/suivi-jetons":
			require.Empty(t, r.URL.RawQuery)
			return response(200, `{"jetons_abonnement":100,"jetons_abonnement_utilises":0,"jetons_pay_as_you_go_restants":2.5}`), nil
		default:
			t.Fatal("route inattendue")
			return nil, nil
		}
	})}))
	suggestions, err := c.Suggestions(context.Background(), " École & associés ", 5)
	require.NoError(t, err)
	require.Len(t, suggestions.Names, 1)
	require.Nil(t, suggestions.Names[0].Name)
	require.Nil(t, suggestions.Names[0].HeadOffice)
	require.Equal(t, "001234567", suggestions.Names[0].SIREN)
	for _, identifier := range []string{"001234567", "00123456700012"} {
		company, err := c.Company(context.Background(), identifier)
		require.NoError(t, err)
		require.Equal(t, "FR12001234567", *company.VATNumber)
		require.Equal(t, "75001", *company.HeadOffice.PostalCode)
	}
	require.NoError(t, c.Ping(context.Background()))
	require.Equal(t, 4, calls)
}

// Les suggestions renvoient l'indicateur de cessation en 0/1, là où la fiche
// entreprise renvoie un vrai booléen, et portent de nombreux champs hors de
// notre projection. Corps synthétique : aucune entreprise réelle en dépôt.
const suggestionsSIRENBody = `{
  "resultats_nom_entreprise": [],
  "resultats_siren": [
    {
      "siren": "001234567",
      "nom_entreprise": "ENTREPRISE TEST",
      "diffusable": true,
      "entreprise_cessee": 0,
      "siege": {
        "siret": "00123456700012",
        "siret_formate": "001 234 567 00012",
        "nic": "00012",
        "numero_voie": 1,
        "indice_repetition": null,
        "type_voie": "RUE",
        "libelle_voie": "TEST",
        "complement_adresse": null,
        "adresse_ligne_1": "1 RUE TEST",
        "adresse_ligne_2": null,
        "code_postal": "75001",
        "code_commune": "75101",
        "ville": "PARIS",
        "pays": "France",
        "code_pays": "FR",
        "latitude": 48.0,
        "longitude": 2.0
      }
    }
  ],
  "resultats_siret": [
    {
      "siren": "001234567",
      "nom_entreprise": "ENTREPRISE TEST",
      "diffusable": true,
      "entreprise_cessee": 0,
      "siege": {"siret": "00123456700012", "ville": "PARIS"}
    }
  ]
}`

func TestSuggestionsCeasedIndicatorAndUnknownFields(t *testing.T) {
	c := New("fixture-token", WithHTTPClient(&http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "001234567", r.URL.Query().Get("q"))
		return response(200, suggestionsSIRENBody), nil
	})}))
	result, err := c.Suggestions(context.Background(), "001234567", 10)
	require.NoError(t, err)
	require.Empty(t, result.Names)
	require.Len(t, result.SIRENs, 1)
	require.Len(t, result.SIRETs, 1)
	for _, company := range []Company{result.SIRENs[0], result.SIRETs[0]} {
		require.Equal(t, "001234567", company.SIREN)
		require.Equal(t, "ENTREPRISE TEST", *company.Name)
		require.Equal(t, "00123456700012", company.HeadOffice.SIRET)
		require.NotNil(t, company.Ceased)
		require.False(t, *company.Ceased)
		encoded, err := json.Marshal(company)
		require.NoError(t, err)
		require.Contains(t, string(encoded), `"entreprise_cessee":false`)
	}
	// La projection ignore les champs Pappers qu'elle ne déclare pas.
	encoded, err := json.Marshal(result.SIRENs[0].HeadOffice)
	require.NoError(t, err)
	for _, absent := range []string{"latitude", "longitude", "nic", "siret_formate", "code_commune"} {
		require.NotContains(t, string(encoded), absent)
	}
}

func TestCompanyCeasedFlagDecoding(t *testing.T) {
	for _, tc := range []struct {
		raw   string
		want  *bool
		valid bool
	}{
		{"true", boolPointer(true), true}, {"false", boolPointer(false), true},
		{"1", boolPointer(true), true}, {"0", boolPointer(false), true},
		{"null", nil, true}, {"", nil, true},
		{`"false"`, nil, false}, {`"0"`, nil, false}, {"2", nil, false},
		{"-1", nil, false}, {"0.5", nil, false}, {"{}", nil, false}, {"[]", nil, false},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			body := `{"siren":"001234567","diffusable":true,"nom_entreprise":"ENTREPRISE TEST","siege":{"siret":"00123456700012"}`
			if tc.raw != "" {
				body += `,"entreprise_cessee":` + tc.raw
			}
			body += "}"
			var company Company
			err := json.Unmarshal([]byte(body), &company)
			if !tc.valid {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, company.Ceased)
			require.Equal(t, "ENTREPRISE TEST", *company.Name)
			require.Equal(t, "00123456700012", company.HeadOffice.SIRET)
			require.True(t, *company.Diffusable)
		})
	}
}

func boolPointer(value bool) *bool { return &value }

func TestInvalidInputsNeverCallProvider(t *testing.T) {
	c := New("secret", WithHTTPClient(&http.Client{Transport: transport(func(*http.Request) (*http.Response, error) { t.Fatal("appel interdit"); return nil, nil })}))
	for _, query := range []string{"", "a", "\n", "a\nb", strings.Repeat("é", 201)} {
		_, err := c.Suggestions(context.Background(), query, 10)
		require.Error(t, err)
	}
	for _, limit := range []int{-1, 0, 101} {
		_, err := c.Suggestions(context.Background(), "test", limit)
		require.Error(t, err)
	}
	for _, id := range []string{"../entreprise", "123", "12345678a", "1234567890"} {
		_, err := c.Company(context.Background(), id)
		require.Error(t, err)
	}
	require.Error(t, New("").Ping(context.Background()))
}

func TestHTTPFailuresAndInvalidResponses(t *testing.T) {
	for _, code := range []int{301, 400, 401, 403, 404, 429, 500, 503} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			calls := 0
			original := &http.Client{Transport: transport(func(*http.Request) (*http.Response, error) {
				calls++
				r := response(code, `{"error":"private-token"}`)
				r.Header.Set("Location", "https://another-host.test/")
				return r, nil
			})}
			c := New("private-token", WithHTTPClient(original))
			err := c.Ping(context.Background())
			var apiErr *APIError
			require.ErrorAs(t, err, &apiErr)
			require.Equal(t, code, apiErr.StatusCode)
			require.NotContains(t, err.Error()+apiErr.Detail(), "private-token")
			require.Equal(t, 1, calls)
			require.Nil(t, original.CheckRedirect)
		})
	}
	for _, body := range []string{"null", "{}", "[]", "<html>private-token</html>", `{"jetons_abonnement":0}`, strings.Repeat("x", maxResponse+1)} {
		c := New("private-token", WithHTTPClient(&http.Client{Transport: transport(func(*http.Request) (*http.Response, error) { return response(200, body), nil })}))
		err := c.Ping(context.Background())
		require.Error(t, err)
		require.NotContains(t, err.Error(), "private-token")
	}
	c := New("private-token", WithHTTPClient(&http.Client{Transport: transport(func(*http.Request) (*http.Response, error) { return response(200, `{"siren":"999999999"}`), nil })}))
	_, err := c.Company(context.Background(), "001234567")
	require.Error(t, err)
	_, err = c.Suggestions(context.Background(), "test", 10)
	require.Error(t, err)
}

func TestCancellationAndNetworkErrors(t *testing.T) {
	c := New("private-token", WithHTTPClient(&http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) { <-r.Context().Done(); return nil, r.Context().Err() })}))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, c.Ping(ctx), context.Canceled)
	ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	require.ErrorIs(t, c.Ping(ctx), context.DeadlineExceeded)
	c = New("private-token", WithHTTPClient(&http.Client{Transport: transport(func(*http.Request) (*http.Response, error) { return nil, errors.New("private-token") })}))
	err := c.Ping(context.Background())
	require.Error(t, err)
	require.NotContains(t, err.Error(), "private-token")
}

func TestWithDoerReceivesAuthenticatedRequestWithoutTokenInURL(t *testing.T) {
	seen := 0
	var observed *http.Request
	doer := func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
		seen++
		observed = req
		return httpClient.Do(req)
	}
	c := New("private-token",
		WithHTTPClient(&http.Client{Transport: transport(func(*http.Request) (*http.Response, error) {
			return response(200, `{"jetons_abonnement":1,"jetons_abonnement_utilises":0,"jetons_pay_as_you_go_restants":1}`), nil
		})}),
		WithDoer(doer),
	)
	require.NoError(t, c.Ping(context.Background()))
	require.Equal(t, 1, seen)
	require.Equal(t, "/v2/suivi-jetons", observed.URL.Path)
	require.Equal(t, "private-token", observed.Header.Get("api-key"))
	require.NotContains(t, observed.URL.String(), "private-token")

	// Un Doer nil laisse le comportement par défaut en place.
	c = New("private-token", WithHTTPClient(&http.Client{Transport: transport(func(*http.Request) (*http.Response, error) {
		return response(200, `{"jetons_abonnement":1,"jetons_abonnement_utilises":0,"jetons_pay_as_you_go_restants":1}`), nil
	})}), WithDoer(nil))
	require.NoError(t, c.Ping(context.Background()))
}
