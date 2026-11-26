package papers_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"strings"
	"testing"

	papers "github.com/ritoon/sdk-papers"
)

func TestClientLoggingRedactsCredentials(t *testing.T) {
	const token = "synthetic-credential-for-logging-test"
	client := papers.New(token)
	values := map[string]any{
		"pointer": client,
		"value":   *client,
		"struct":  struct{ Client papers.Client }{*client},
		"slice":   []papers.Client{*client},
		"map":     map[string]any{"client": client},
	}
	check := func(t *testing.T, output string) {
		t.Helper()
		if strings.Contains(output, token) || strings.Contains(strings.ToLower(output), hex.EncodeToString([]byte(token))) {
			t.Fatal("client logging exposed the synthetic credential")
		}
	}
	for name, value := range values {
		t.Run(name, func(t *testing.T) {
			for _, format := range []string{"%v", "%+v", "%#v", "%s", "%q", "%x", "%X", "%d", "%+100.10v"} {
				t.Run(format, func(t *testing.T) { check(t, fmt.Sprintf(format, value)) })
			}
			var output bytes.Buffer
			log.New(&output, "", 0).Print(value)
			check(t, output.String())
			for _, handler := range []slog.Handler{
				slog.NewTextHandler(&output, nil),
				slog.NewJSONHandler(&output, nil),
			} {
				output.Reset()
				slog.New(handler).Info("client", "client", value)
				check(t, output.String())
			}
			encoded, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			check(t, string(encoded))
		})
	}
}

func TestClientLoggingRepresentation(t *testing.T) {
	const want = "papers.Client{token:[REDACTED]}"
	for _, client := range []papers.Client{*papers.New("synthetic-secret"), {}} {
		if client.String() != want || client.LogValue().String() != want {
			t.Fatal("expected a fully redacted representation")
		}
		for _, value := range []any{client, &client} {
			if fmt.Sprintf("%#v", value) != want {
				t.Fatal("expected redacted Go syntax formatting")
			}
			var output bytes.Buffer
			slog.New(slog.NewJSONHandler(&output, nil)).Info("client", "client", value)
			var entry map[string]any
			if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
				t.Fatal(err)
			}
			if entry["client"] != want {
				t.Fatal("expected redacted structured logging")
			}
		}
	}
}
