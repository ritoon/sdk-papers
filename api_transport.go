package papers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"
)

// Response conserve le statut HTTP (notamment 206), les en-têtes et le corps décodé.
// Body contient tous les champs décrits par le contrat ; RawBody conserve aussi
// les éventuelles extensions du fournisseur. Ne pas journaliser ces données.
type Response[T any] struct {
	StatusCode int
	Header     map[string][]string
	Body       T
	RawBody    []byte
}

type responseKind int

const (
	responseJSON responseKind = iota
	responseBinary
	responseEmpty
)

// Pointer facilite les options facultatives, y compris false et zéro.
func Pointer[T any](value T) *T { return &value }

func queryValues(params any) (url.Values, error) {
	values := url.Values{}
	v := reflect.ValueOf(params)
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		tag := strings.Split(v.Type().Field(i).Tag.Get("query"), ",")
		if field.IsNil() {
			if len(tag) > 1 {
				return nil, errors.New("paramètre Papers obligatoire : " + tag[0])
			}
			continue
		}
		values.Set(tag[0], fmt.Sprint(field.Elem().Interface()))
	}
	return values, nil
}

func callAPI[T any](c *Client, ctx context.Context, method, path string, params, body any, kind responseKind, statuses []int) (*Response[T], error) {
	query, err := queryValues(params)
	if err != nil {
		return nil, err
	}
	var payload []byte
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			return nil, errors.New("corps Papers JSON invalide")
		}
	}
	data, status, header, err := c.request(ctx, method, path, query, payload, kind == responseBinary, statuses)
	if err != nil {
		return nil, err
	}
	result := &Response[T]{StatusCode: status, Header: header, RawBody: data}
	switch kind {
	case responseBinary:
		// Only generated binary operations use T=[]byte.
		result.Body = any(data).(T)
	case responseJSON:
		if bytes.Equal(bytes.TrimSpace(data), []byte("null")) || json.Unmarshal(data, &result.Body) != nil {
			return nil, errors.New("réponse Papers JSON invalide")
		}
	}
	return result, nil
}
