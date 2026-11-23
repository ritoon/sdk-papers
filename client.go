// Package papers fournit un client Go en lecture seule pour l'API Pappers, le
// registre des entreprises françaises : autocomplétion, fiche entreprise et
// suivi des crédits.
//
// Le client n'expose que des lectures : aucune création ni modification côté
// Pappers. Le jeton est transmis dans l'en-tête api-key, uniquement vers
// https://api.pappers.fr/v2, et n'apparaît jamais dans une URL ni dans une
// erreur. Les redirections sont refusées, aucune tentative n'est rejouée, et la
// réponse est bornée en taille comme en durée.
package papers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const baseURL = "https://api.pappers.fr/v2"
const maxResponse = 4 << 20

type Client struct {
	token string
	http  *http.Client
	doer  Doer
}

// Doer exécute la requête préparée par le client. Le point d'extension permet
// d'observer ou d'instrumenter les échanges (journalisation, capture de
// diagnostic, métriques) sans donner accès au jeton : il reçoit la requête déjà
// signée et doit la transmettre telle quelle.
type Doer func(*http.Client, *http.Request) (*http.Response, error)

func defaultDoer(httpClient *http.Client, req *http.Request) (*http.Response, error) {
	return httpClient.Do(req)
}

type Option func(*Client)

// WithDoer remplace l'exécution de la requête. Un Doer nil est ignoré.
func WithDoer(doer Doer) Option {
	return func(c *Client) {
		if doer != nil {
			c.doer = doer
		}
	}
}

// WithHTTPClient permet notamment d'injecter un transport pour les tests.
// Le client est copié et les redirections restent toujours interdites.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		if httpClient != nil {
			copy := *httpClient
			c.http = &copy
		}
	}
}

func New(token string, opts ...Option) *Client {
	c := &Client{token: strings.TrimSpace(token), http: &http.Client{}, doer: defaultDoer}
	for _, opt := range opts {
		opt(c)
	}
	c.http.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	return c
}

// APIError n'inclut jamais le corps distant ni les identifiants de la requête.
type APIError struct{ StatusCode int }

func (e *APIError) Error() string { return fmt.Sprintf("Papers : HTTP %d", e.StatusCode) }

func (e *APIError) Detail() string {
	switch e.StatusCode {
	case 400:
		return "Paramètres refusés par Pappers."
	case 401:
		return "Clé API incorrecte."
	case 402, 403:
		return "Accès refusé ou crédits insuffisants. Vérifiez l’offre Pappers."
	case 404:
		return "Entreprise introuvable."
	case 429:
		return "Limite d’appels atteinte. Réessayez plus tard."
	default:
		return "Service Pappers indisponible ou réponse inattendue."
	}
}

var identifierPattern = regexp.MustCompile(`^(?:[0-9]{9}|[0-9]{14})$`)

func ValidIdentifier(value string) bool { return identifierPattern.MatchString(value) }

func ValidQuery(value string) bool {
	n := utf8.RuneCountInString(value)
	return utf8.ValidString(value) && n >= 2 && n <= 200 && strings.IndexFunc(value, unicode.IsControl) < 0
}

// Suggestions recherche seulement les entreprises, par nom, SIREN et SIRET.
func (c *Client) Suggestions(ctx context.Context, query string, limit int) (*Suggestions, error) {
	query = strings.TrimSpace(query)
	if !ValidQuery(query) || limit < 1 || limit > 100 {
		return nil, errors.New("recherche Papers invalide")
	}
	var result Suggestions
	err := c.get(ctx, "/suggestions", url.Values{"q": {query}, "longueur": {strconv.Itoa(limit)}, "cibles": {"nom_entreprise,siren,siret"}}, &result)
	if err != nil {
		return nil, err
	}
	// Un objet sans les listes attendues ne prouve pas la réussite de l'appel.
	if result.Names == nil || result.SIRENs == nil || result.SIRETs == nil {
		return nil, errors.New("réponse Papers invalide")
	}
	return &result, nil
}

func (c *Client) Company(ctx context.Context, identifier string) (*Company, error) {
	if !ValidIdentifier(identifier) {
		return nil, errors.New("SIREN ou SIRET invalide")
	}
	key := "siren"
	if len(identifier) == 14 {
		key = "siret"
	}
	var result Company
	if err := c.get(ctx, "/entreprise", url.Values{key: {identifier}}, &result); err != nil {
		return nil, err
	}
	if result.SIREN != identifier[:9] {
		return nil, errors.New("réponse Papers incohérente avec le SIREN demandé")
	}
	if key == "siret" && (result.Establishment == nil || result.Establishment.SIRET != identifier) {
		return nil, errors.New("réponse Papers incohérente avec le SIRET demandé")
	}
	return &result, nil
}

func (c *Client) Usage(ctx context.Context) (*Usage, error) {
	var result Usage
	if err := c.get(ctx, "/suivi-jetons", nil, &result); err != nil {
		return nil, err
	}
	if result.Subscription == nil || result.Used == nil || result.Remaining == nil {
		return nil, errors.New("suivi Papers invalide")
	}
	return &result, nil
}

// Ping utilise une route authentifiée : les suggestions peuvent être publiques
// et ne suffiraient donc pas à valider le jeton.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.Usage(ctx)
	return err
}

func (c *Client) get(parent context.Context, path string, query url.Values, result any) error {
	if c.token == "" {
		return errors.New("Papers non configuré")
	}
	ctx, cancel := context.WithTimeout(parent, 25*time.Second)
	defer cancel()
	u := baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return errors.New("requête Papers invalide")
	}
	req.Header.Set("api-key", c.token)
	req.Header.Set("Accept", "application/json")
	res, err := c.doer(c.http, req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		var networkErr net.Error
		if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &networkErr) && networkErr.Timeout()) {
			return context.DeadlineExceeded
		}
		return errors.New("connexion Papers impossible")
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return &APIError{StatusCode: res.StatusCode}
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, maxResponse+1))
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("réponse Papers illisible")
	}
	if len(data) > maxResponse {
		return errors.New("réponse Papers trop volumineuse")
	}
	if json.Unmarshal(data, result) != nil {
		return errors.New("réponse Papers JSON invalide")
	}
	return nil
}
