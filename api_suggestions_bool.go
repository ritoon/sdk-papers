package papers

import (
	"bytes"
	"errors"
)

// Boolean01 est un booléen JSON qui accepte aussi les indicateurs numériques
// 0 et 1 observés sur les suggestions Pappers. La sérialisation reste booléenne.
// Les champs *Boolean01 conservent null grâce au décodage standard des pointeurs.
type Boolean01 bool

func (value *Boolean01) UnmarshalJSON(data []byte) error {
	switch string(bytes.TrimSpace(data)) {
	case "true", "1":
		*value = true
	case "false", "0":
		*value = false
	default:
		return errors.New("Papers : indicateur booléen invalide")
	}
	return nil
}
