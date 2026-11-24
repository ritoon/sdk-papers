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
| `DownloadINSEENotice` | `GET /document/avis_situation_insee` |
| `DownloadArticles` | `GET /document/statuts` |
| `DownloadBeneficiaryDeclaration` | `GET /document/declaration_beneficiaires_effectifs` |
| `DownloadFinancialReport` | `GET /document/rapport_financier` |
| `DownloadNonFinancialReport` | `GET /document/rapport_non_financier` |
| `CheckPersonCompliance` | `GET /conformite/personne_physique` |
| `WatchCompanies` | `POST /liste` |
| `WatchOfficers` | `POST /liste/` |
| `DeleteNotifications` | `DELETE /liste/` |
| `AddNotificationInformation` | `POST /liste-informations` |
| `DecodeCompanyWebhook` | `WEBHOOK webhook_entreprise` |
| `DecodeOfficerWebhook` | `WEBHOOK webhook_dirigeant` |
| `DecodeMonitoringWebhook` | `WEBHOOK veille_evenement` |

## Recherche et pagination

```go
result, err := client.SearchCompanies(ctx, papers.SearchCompaniesParams{
    Q: papers.Pointer("entreprise fictive"),
    StatutRcs: papers.Pointer("Radié"),
    ParPage: papers.Pointer(int64(10)),
    Page: papers.Pointer(int64(1)),
})
// result.Body.Resultats, result.Body.Total et result.Body.Page sont accessibles.
// Pour la pagination par curseur, transmettre Curseur puis lire CurseurSuivant.
```

`SuggestionsWithOptions` donne accès aux six cibles, n'impose pas la limite de
2–200 caractères de la méthode historique et accepte les indicateurs de
cessation `0`/`1` ou booléens (`Boolean01`). Une clé API reste requise, conformément
à la sécurité héritée du contrat OpenAPI, même si le service peut permettre des
suggestions publiques. Les cibles multiples sont séparées par des virgules.

## Documents et surveillance

Les méthodes `Download…` retournent `Response[[]byte]` : utiliser `Body` pour le
contenu PDF/XLSX et les en-têtes pour le type de contenu et le nom du fichier.
Un document de plus de 4 Mio est rejeté par la limite de transport du SDK.

`WatchCompanies`, `WatchOfficers`, `DeleteNotifications` et
`AddNotificationInformation` modifient les listes Pappers lorsqu'on les appelle.
Les ajouts acceptent 200 et 201, les informations acceptent un 200 vide ; les
routes `/liste` et `/liste/` restent distinctes. Aucun de ces appels n'est
exécuté lors de la construction du client ou par les tests.

Le contrat des dirigeants marque `siren` obligatoire tout en décrivant les
personnes physiques avec nom, prénom et date de naissance. Le DTO conserve les
deux formes sans inventer une restriction locale : la validation métier reste
au fournisseur. Cet écart documentaire demande une recette sur compte réel.

## Webhooks entrants

Les fonctions `DecodeCompanyWebhook`, `DecodeOfficerWebhook` et
`DecodeMonitoringWebhook` décodent le corps fourni par votre serveur HTTP. Elles
vérifient le type d'événement, la présence de `donnees`, le JSON et la limite de
4 Mio ; elles ne déploient pas de serveur et n'authentifient pas l'émetteur.
L'application doit limiter le corps HTTP avant de le lire (par exemple avec
`http.MaxBytesReader`), vérifier son origine selon sa configuration et gérer les
doublons. Aucune signature non documentée n'est supposée.

Les événements et leurs notifications sont typés. Les champs supplémentaires
restent accessibles dans le corps d'entrée, conservé par l'application.
