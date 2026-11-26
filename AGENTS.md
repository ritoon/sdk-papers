# Consignes pour les agents

## Exploration du code

- Privilégier `rg --files` pour repérer les fichiers et `rg -l` pour identifier ceux qui contiennent un symbole. Restreindre la recherche aux chemins ou types de fichiers pertinents.
- Passer ensuite à `rg -n` avec un motif précis et lire uniquement les extraits utiles. Si la sortie est volumineuse, affiner le motif ou limiter le nombre de résultats affichés.
- Lors d’une recherche générale, écarter d’abord les gros fichiers générés lorsqu’ils masquent les résultats utiles. Les inclure explicitement si la recherche porte sur un type ou un contrat généré.
- Si `rg` est indisponible, utiliser l’outil de recherche disponible avec les mêmes limites de périmètre et de sortie.
