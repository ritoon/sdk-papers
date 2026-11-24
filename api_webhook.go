package papers

import (
	"bytes"
	"encoding/json"
	"errors"
)

// decodeWebhook validates the envelope, not the sender. The application must
// authenticate incoming requests before calling a decoder and limit its HTTP
// request body (for example with http.MaxBytesReader).
func decodeWebhook[T any](data []byte, event string) (*T, error) {
	if len(data) > maxResponse {
		return nil, errors.New("webhook Papers trop volumineux")
	}
	var envelope struct {
		Event string          `json:"evenement"`
		Data  json.RawMessage `json:"donnees"`
	}
	if json.Unmarshal(data, &envelope) != nil || envelope.Event != event || len(envelope.Data) == 0 || bytes.Equal(bytes.TrimSpace(envelope.Data), []byte("null")) {
		return nil, errors.New("webhook Papers invalide")
	}
	var value T
	if json.Unmarshal(data, &value) != nil {
		return nil, errors.New("webhook Papers invalide")
	}
	return &value, nil
}
