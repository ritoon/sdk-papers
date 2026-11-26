# Consignes pour les agents

## Exploration du code

- Privilégier `rg --files` pour repérer les fichiers et `rg -l` pour identifier ceux qui contiennent un symbole. Restreindre la recherche aux chemins ou types de fichiers pertinents.
- Passer ensuite à `rg -n` avec un motif précis et lire uniquement les extraits utiles. Si la sortie est volumineuse, affiner le motif ou limiter le nombre de résultats affichés.
- Lors d’une recherche générale, écarter d’abord les gros fichiers générés lorsqu’ils masquent les résultats utiles. Les inclure explicitement si la recherche porte sur un type ou un contrat généré.
- Si `rg` est indisponible, utiliser l’outil de recherche disponible avec les mêmes limites de périmètre et de sortie.

## Utiliser et maintenir CodeGraph

- **Choisir l’outil selon la question.** Dans ce dépôt, CodeGraph n’est pas
  un préalable aux recherches ou aux lectures. La présence d’un index et les
  recommandations génériques « CodeGraph d’abord » ne rendent pas son appel
  obligatoire : les règles de choix ci-dessous s’appliquent.
- **Fichier, symbole connu ou texte précis :** utiliser `rg` puis lire la plage
  utile ; lire directement cette plage si son emplacement est déjà connu.
  Utiliser le LSP/`gopls` pour une définition ou des références si disponible.
- **Appelants, dépendances et impact d’un changement :** privilégier une requête
  de graphe ciblée (`codegraph callers <symbole>`, `codegraph callees <symbole>`
  ou `codegraph impact <symbole>`), lorsque l’index est exploitable.
- **Flux complexe entre plusieurs fichiers :** utiliser `codegraph_explore`
  ou `codegraph explore --max-files 3 "<symboles du flux>"` si les relations
  sont nécessaires à la réponse. Côté MCP, fixer `maxFiles` à 3 au départ.
  Nommer les symboles pertinents, puis élargir seulement pour une dépendance
  manquante. `maxFiles` limite les fichiers de source, pas tout le texte renvoyé ;
  un dossier cité dans la requête n’est pas un filtre strict de périmètre.
- **Limiter le contexte :** réutiliser les extraits déjà obtenus et encore
  valides. Éviter de relire un fichier déjà fourni par le graphe ; vérifier
  toutefois le code courant si la branche ou le fichier a changé. Ne pas lancer
  une exploration générale pour localiser une seule fonction connue.
- **Index et cache :** l’index local accélère la recherche des relations ; il ne
  supprime pas les tokens du texte retourné au modèle. Le cache du modèle est
  distinct et peut aussi bénéficier aux lectures classiques. Ne pas promettre
  une économie de tokens ou de coût sans mesure du parcours complet.
- **Index absent ou inutilisable :** poursuivre avec `rg`, le LSP et des lectures
  ciblées. L’indexation initiale (`codegraph init`) reste un choix explicite de
  l’utilisateur, pas un prérequis déclenché automatiquement pour une recherche.
- Pour un index existant, vérifier sa disponibilité et sa fraîcheur avec
  `codegraph status`. Le watcher le maintient normalement à jour ; lancer
  `codegraph sync` en cas de retard. Si l’index appartient à un autre checkout,
  recouper les résultats importants avec la branche de travail. Garder les
  fichiers d’index hors de Git.
- Pour vérifier si une version du CLI est disponible, utiliser
  `codegraph upgrade --check`. Après un `codegraph upgrade`, redémarrer la
  session MCP si elle utilise encore l’ancien processus, puis lancer
  `codegraph index` et `codegraph status`. En dehors d’une mise à jour, réserver
  la reconstruction complète aux problèmes persistants après synchronisation.

Voir la [référence CLI CodeGraph](https://github.com/colbymchenry/codegraph/blob/main/site/src/content/docs/reference/cli.md).
