package papers

import (
	"bytes"
	"encoding/json"
	"errors"
)

// Projection des données d'identité utiles à la création d'entreprise.
// Les champs potentiellement absents restent null (diffusion partielle notamment).
type Company struct {
	SIREN         string         `json:"siren"`
	Name          *string        `json:"nom_entreprise"`
	Denomination  *string        `json:"denomination"`
	VATNumber     *string        `json:"numero_tva_intracommunautaire"`
	LegalForm     *string        `json:"forme_juridique"`
	NAFCode       *string        `json:"code_naf"`
	NAFLabel      *string        `json:"libelle_code_naf"`
	Diffusable    *bool          `json:"diffusable"`
	Ceased        *bool          `json:"entreprise_cessee"`
	HeadOffice    *Establishment `json:"siege"`
	Establishment *Establishment `json:"etablissement,omitempty"`
}

// UnmarshalJSON accepte aussi les indicateurs 0/1 renvoyés par les suggestions
// Pappers, là où la fiche entreprise renvoie un vrai booléen. Le contrat exposé
// par ce paquet reste un booléen nullable dans les deux cas.
func (c *Company) UnmarshalJSON(data []byte) error {
	type company Company
	var decoded company
	wire := struct {
		*company
		Ceased json.RawMessage `json:"entreprise_cessee"`
	}{company: &decoded}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	switch string(bytes.TrimSpace(wire.Ceased)) {
	case "", "null":
	case "false", "0":
		value := false
		decoded.Ceased = &value
	case "true", "1":
		value := true
		decoded.Ceased = &value
	default:
		return errors.New("Papers : entreprise_cessee doit être un booléen, 0, 1 ou null")
	}
	*c = Company(decoded)
	return nil
}

type Establishment struct {
	SIRET       string  `json:"siret"`
	Address1    *string `json:"adresse_ligne_1"`
	Address2    *string `json:"adresse_ligne_2"`
	PostalCode  *string `json:"code_postal"`
	City        *string `json:"ville"`
	Country     *string `json:"pays"`
	CountryCode *string `json:"code_pays"`
}

type Suggestions struct {
	Names  []Company `json:"resultats_nom_entreprise"`
	SIRENs []Company `json:"resultats_siren"`
	SIRETs []Company `json:"resultats_siret"`
}

type Usage struct {
	Subscription *float64 `json:"jetons_abonnement"`
	Used         *float64 `json:"jetons_abonnement_utilises"`
	Remaining    *float64 `json:"jetons_pay_as_you_go_restants"`
}
