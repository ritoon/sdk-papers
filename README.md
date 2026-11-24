# sdk-papers

Client Go pour l’[API Pappers](https://www.pappers.fr/api),
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
les métriques. Ce callback reçoit le jeton dans l’en-tête `api-key` : il doit
être considéré comme du code de confiance, tout comme un transport personnalisé.
Ne journalisez pas les en-têtes ni les requêtes/réponses complètes : masquez
`api-key` et évitez de capturer des données réelles. L’exemple ci-dessous ne
journalise que la méthode, le chemin sans paramètres et la durée :

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
  `https://api.pappers.fr/v2` avec le transport par défaut. Le SDK ne le place
  jamais dans une URL ni dans les erreurs qu’il renvoie. Un `Doer` ou un transport
  personnalisé a accès au jeton et reste responsable de son utilisation.
- Les méthodes historiques restent en lecture seule. Les nouvelles opérations
  exposent explicitement les options du contrat ; les options payantes ne sont
  jamais activées automatiquement.
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

La CI vérifie le formatage, `go vet` et les tests avec le détecteur de courses
concurrentes, sur Go 1.24 et la version stable. Elle échoue si la couverture
globale des instructions descend sous **99 %**, et conserve le profil de
couverture comme artefact. Pour reproduire ce contrôle localement :

```bash
go test -race -count=1 -covermode=atomic -coverpkg=./... -coverprofile=/tmp/sdk-papers-coverage.out ./...
sh scripts/check-coverage.sh /tmp/sdk-papers-coverage.out
go tool cover -func=/tmp/sdk-papers-coverage.out
```

Ce seuil mesure les instructions exécutées, pas toutes les combinaisons de
branches ni la compatibilité avec le service réel.

Les tests n’appellent pas Pappers : les transports sont simulés et les corps de
réponse sont synthétiques. Sont couverts l’authentification, l’encodage des
paramètres, les champs absents, le décodage `0`/`1`, les champs hors
projection, les crédits à zéro, les délais, l’annulation, les redirections, les
codes d’erreur, les limites de taille et le non-divulgation du jeton.

## Prévention des secrets et données réelles

Les exemples et fixtures doivent rester synthétiques : ne pas copier de réponses
Pappers réelles, de données clients ou de jetons valides dans le dépôt.
Les fichiers `.env` et `.env.*` sont ignorés ; seul `.env.example` peut être suivi
et doit contenir uniquement des valeurs factices. Une exclusion Git ne protège
pas un fichier déjà suivi.

La CI recherche les secrets dans tout l’historique accessible avec
[Gitleaks v8.24.3](https://github.com/gitleaks/gitleaks/releases/tag/v8.24.3).
Pour lancer le même contrôle localement depuis la racine du dépôt :

```bash
go install github.com/zricethezav/gitleaks/v8@v8.24.3
"$(go env GOPATH)/bin/gitleaks" git --redact=100 --log-opts="--all" .
```

Le clone doit contenir l’historique à contrôler (`git fetch --unshallow` pour un
clone superficiel). Pour examiner également les fichiers avant commit :

```bash
"$(go env GOPATH)/bin/gitleaks" dir --redact=100 .
```

Une détection fait échouer le contrôle ; les valeurs sont masquées dans la sortie.
Aucune exclusion globale des tests n’est appliquée. Tout faux positif doit faire
l’objet d’une exception précise, justifiée en revue. Ce scan ne garantit ni
l’absence de toutes les clés API ni la détection des données personnelles ou
métier : la revue des exemples reste nécessaire.

## Référence

[Documentation de l’API Pappers](https://www.pappers.fr/api/documentation) —
contrat v2.

## Licence

[MIT](LICENSE).

## Couverture étendue Pappers 2.20.0

Les opérations supplémentaires sont décrites dans [api/README.md](api/README.md).
Les méthodes historiques `Company`, `Suggestions`, `Usage` et `Ping` restent
compatibles. `CompanyDetails` expose la fiche complète et ses options, y compris
les statuts RCS/RNE, les dates de radiation et les procédures collectives.
