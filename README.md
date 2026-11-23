# sdk-papers

Client Go en lecture seule pour l’[API Pappers](https://www.pappers.fr/api),
le registre des entreprises françaises : autocomplétion, fiche entreprise et
suivi des crédits.

Sans dépendance hors bibliothèque standard (`testify` uniquement pour les tests).

```bash
go get github.com/ritoon/sdk-papers
```

## Démarrage

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	papers "github.com/ritoon/sdk-papers"
)

func main() {
	client := papers.New(os.Getenv("PAPPERS_API_TOKEN"))

	if err := client.Ping(context.Background()); err != nil {
		log.Fatalf("jeton Pappers refusé : %v", err)
	}

	// SIREN de 9 chiffres ou SIRET de 14 chiffres.
	company, err := client.Company(context.Background(), "123456789")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(*company.Name, company.SIREN)
}
```

## Lectures disponibles

| Méthode | Route Pappers | Usage |
|---|---|---|
| `Suggestions(ctx, query, limit)` | `GET /suggestions` | Recherche par nom, SIREN et SIRET ; requête de 2 à 200 caractères, 1 à 100 résultats par cible |
| `Company(ctx, identifier)` | `GET /entreprise` | SIREN de 9 chiffres ou SIRET de 14 chiffres ; identité, TVA et adresse |
| `Usage(ctx)` | `GET /suivi-jetons` | Crédits d’abonnement consommés et restants |
| `Ping(ctx)` | `GET /suivi-jetons` | Vérification du jeton |

`Ping` passe par le suivi des crédits, et non par les suggestions : celles-ci
sont accessibles publiquement chez Pappers et ne prouveraient donc pas que le
jeton est valide.

Les helpers `ValidQuery` et `ValidIdentifier` permettent de valider une saisie
avant appel ; le client les applique de toute façon et n’émet aucune requête
réseau sur une entrée invalide.

## Options

```go
client := papers.New(token,
	papers.WithHTTPClient(httpClient), // transport personnalisé (le client est copié)
	papers.WithDoer(myDoer),           // observation des échanges
)
```

`WithDoer` reçoit la requête déjà signée et doit la transmettre telle quelle.
C’est le point d’extension pour la journalisation, la capture de diagnostic ou
les métriques, sans donner accès au jeton :

```go
func myDoer(httpClient *http.Client, req *http.Request) (*http.Response, error) {
	start := time.Now()
	res, err := httpClient.Do(req)
	log.Printf("%s %s en %s", req.Method, req.URL.Path, time.Since(start))
	return res, err
}
```

Un `Doer` nil est ignoré et laisse le comportement par défaut en place. Quelle
que soit l’option, les redirections restent interdites : `CheckRedirect` est
réinstallé après application des options et le client fourni n’est pas modifié.

## Erreurs

Une réponse HTTP non-200 produit une `*papers.APIError` portant le code :

```go
var apiErr *papers.APIError
if errors.As(err, &apiErr) {
	fmt.Println(apiErr.StatusCode, apiErr.Detail())
}
```

`Detail()` renvoie un message court destiné à l’affichage (clé incorrecte,
crédits insuffisants, entreprise introuvable, limite d’appels atteinte…).
Ni `Error()` ni `Detail()` ne contiennent le corps distant ou le jeton.

Une annulation ou un dépassement de délai remonte `context.Canceled` ou
`context.DeadlineExceeded`, jamais une erreur réseau brute.

## Garanties

- Le jeton est envoyé dans l’en-tête `api-key`, uniquement vers
  `https://api.pappers.fr/v2`. Jamais dans une URL, jamais dans une erreur.
- Lecture seule : aucune création ni modification côté Pappers, aucun
  enrichissement payant ni vérification VIES demandés.
- Redirections refusées, aucune tentative rejouée automatiquement.
- Délai maximum de 25 secondes par appel, réponse limitée à 4 Mio.
- Les réponses sont contrôlées avant d’être rendues : une fiche dont le SIREN
  ou le SIRET ne correspond pas à l’identifiant demandé est rejetée, et des
  suggestions dépourvues des trois listes attendues le sont aussi.

## Modèle de données

Les DTO projettent les champs utiles à une fiche entreprise en conservant les
noms JSON de Pappers. Points à connaître :

- Les champs absents ou en diffusion partielle restent `null` — d’où les
  pointeurs sur presque tous les champs.
- `entreprise_cessee` arrive en booléen sur la fiche entreprise mais en `0`/`1`
  sur les suggestions. `Company.UnmarshalJSON` normalise les deux, et conserve
  `null` tel quel ; toute autre valeur est une erreur.
- Le numéro de TVA intracommunautaire est celui renvoyé par Pappers. Il n’est
  **pas** déduit du SIREN.
- Une recherche par SIRET renvoie à la fois le siège (`siege`) et
  l’établissement demandé (`etablissement`).
- Les mêmes entreprises peuvent figurer dans plusieurs des trois listes de
  suggestions : la déduplication est à la charge de l’appelant.

## Tests

```bash
go test ./...
```

Les tests n’appellent pas Pappers : les transports sont simulés et les corps de
réponse sont synthétiques. Sont couverts l’authentification, l’encodage des
paramètres, les champs absents, le décodage `0`/`1`, les champs hors
projection, les crédits à zéro, les délais, l’annulation, les redirections, les
codes d’erreur, les limites de taille et le non-divulgation du jeton.

## Référence

[Documentation de l’API Pappers](https://www.pappers.fr/api/documentation) —
contrat v2.

## Licence

[MIT](LICENSE).
