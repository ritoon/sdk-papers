# Contrat Pappers 2.20.0

Source : https://www.pappers.fr/api_v2.yaml?20260928, consultée le 24 novembre
2026. `pappers-2.20.0.json` est une conversion JSON du contrat officiel : seuls
les exemples ont été supprimés pour ne pas introduire de données réelles dans
les fixtures. Les tests utilisent exclusivement des valeurs synthétiques.

`operations.json` active les méthodes dans l'ordre de leur introduction.
Régénération reproductible, sans réseau ni dépendance Python :

```sh
python3 scripts/generate-api.py
gofmt -w api_*.go
```

Les types conservent les noms métier français du contrat, avec des noms de
champs Go exportés. Les champs scalaires sont des pointeurs pour représenter
les valeurs absentes ou nulles. Les dates restent des chaînes : le fournisseur
décrit plusieurs formats. Les propriétés sans schéma précis sont conservées en
`json.RawMessage` ; les objets sans propriétés déclarées restent des maps.
Les paramètres sont typés et facultatifs via des pointeurs (`papers.Pointer`).
Le client contrôle la présence des paramètres marqués obligatoires dans
OpenAPI. Les règles métier, plages, combinaisons et droits sont validés par
Pappers ; aucun filtre ni enrichissement n'est ajouté automatiquement.

Chaque méthode étendue retourne `Response[T]` avec `Body`, `StatusCode`,
`Header` et `RawBody`. Ce dernier préserve également les extensions non décrites
par le contrat. HTTP 206 est un succès partiel explicite pour `CompanyDetails`.
Les réponses et paramètres ne doivent pas être journalisés sans filtrage.
Le transport conserve l'authentification par en-tête, l'interdiction des
redirections, le délai de 25 secondes et la limite de 4 Mio par réponse,
y compris pour les documents. Aucun retry automatique ni pagination automatique.

```go
result, err := client.CompanyDetails(ctx, papers.CompanyDetailsParams{
    Siren: papers.Pointer("001234567"),
    ValiditeTvaIntracommunautaire: papers.Pointer(false),
})
// Vérifier err, puis result.StatusCode (200 ou 206) et result.Body.StatutRcs.
```

La couverture documentaire et les tests simulés ne constituent pas une recette
sur un compte réel. Les méthodes `Company` et `Suggestions` historiques gardent
leur projection et leurs restrictions ; utiliser les variantes étendues pour
accéder au contrat complet.

## Opérations

| Méthode | Route / événement |
|---|---|
| `CompanyDetails` | `GET /entreprise` |
| `AssociationDetails` | `GET /association` |
| `SearchCompanies` | `GET /recherche` |
| `SearchOfficers` | `GET /recherche-dirigeants` |
| `SearchBeneficiaries` | `GET /recherche-beneficiaires` |
| `SearchDocuments` | `GET /recherche-documents` |
| `SearchPublications` | `GET /recherche-publications` |
| `SuggestionsWithOptions` | `GET /suggestions` |
| `AnnualAccounts` | `GET /entreprise/comptes` |
| `CompanyNetwork` | `GET /entreprise/cartographie` |
| `DownloadDocument` | `GET /document/telechargement` |
| `DownloadPappersExtract` | `GET /document/extrait_pappers` |
| `DownloadINPIExtract` | `GET /document/extrait_inpi` |
