#!/bin/sh
# Compare les nombres d’instructions, sans arrondir le taux avant le contrôle.
set -eu
awk '
NR == 1 {
    if ($0 !~ /^mode: (set|count|atomic)$/) exit 2
    next
}
NF != 3 || $2 !~ /^[0-9]+$/ || $3 !~ /^[0-9]+$/ { invalid = 1; next }
{
    total += $2
    if ($3 > 0) covered += $2
}
END {
    if (invalid || total == 0) {
        print "Profil de couverture invalide ou vide."
        exit 1
    }
    printf "Couverture : %.2f %% (%d/%d instructions), minimum : 99 %%\n", 100 * covered / total, covered, total
    if (covered * 100 < total * 99) exit 1
}' "${1:?Usage: sh scripts/check-coverage.sh coverage.out}"
