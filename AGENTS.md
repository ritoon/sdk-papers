# Consignes pour les agents

## Exploration du code

- Privilégier `rg --files` pour repérer les fichiers et `rg -l` pour identifier ceux qui contiennent un symbole. Restreindre la recherche aux chemins ou types de fichiers pertinents.
- Passer ensuite à `rg -n` avec un motif précis et lire uniquement les extraits utiles. Si la sortie est volumineuse, affiner le motif ou limiter le nombre de résultats affichés.
- Lors d’une recherche générale, écarter d’abord les gros fichiers générés lorsqu’ils masquent les résultats utiles. Les inclure explicitement si la recherche porte sur un type ou un contrat généré.
- Si `rg` est indisponible, utiliser l’outil de recherche disponible avec les mêmes limites de périmètre et de sortie.

## Utiliser et maintenir CodeGraph

- Garder `rg` pour le texte et les chemins. Pour suivre les appelants, les
  dépendances ou le rayon d’impact d’un symbole, utiliser `codegraph callers`,
  `callees` ou `impact`. Pour suivre un flux, cibler les symboles avec
  `codegraph explore --max-files 3 "<symboles>"` ou une requête MCP ciblée.
- Depuis la racine du dépôt, utiliser `codegraph init` une fois si l’index est
  absent. Le watcher maintient normalement l’index à jour. Vérifier avec
  `codegraph status`, puis lancer `codegraph sync` si l’index a pris du retard.
  Si l’index est celui d’un autre checkout, vérifier les résultats importants
  sur la branche de travail. Garder les fichiers d’index hors de Git.
- Pour vérifier si une version du CLI est disponible, utiliser
  `codegraph upgrade --check`. Après un `codegraph upgrade`, redémarrer la
  session MCP si elle utilise encore l’ancien processus, puis lancer
  `codegraph index` et `codegraph status`. En dehors d’une mise à jour, réserver
  la reconstruction complète aux problèmes persistants après synchronisation.

Voir la [référence CLI CodeGraph](https://github.com/colbymchenry/codegraph/blob/main/site/src/content/docs/reference/cli.md).
