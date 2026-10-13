package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchGraphQLPayload(t *testing.T) {
	const expectedQuery = `{
            grid {totalSlots, maxSession, sessionCount, sessionQueueSize, nodeCount, version },
            nodesInfo { nodes { id, uri, status, maxSession, slotCount, sessionCount, version, stereotypes } }
        }`
	const response = `{"data":{"grid":{"totalSlots":8},"nodesInfo":{"nodes":[]}}}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/graphql" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if !json.Valid(body) {
			t.Errorf("request body is not valid JSON: %q", body)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if !bytes.Contains(body, []byte(`\n`)) {
			t.Errorf("request body has no escaped newlines: %q", body)
		}
		for i, b := range body {
			if b < 0x20 {
				t.Errorf("raw control character 0x%02x at byte %d in serialized payload", b, i)
			}
		}

		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Errorf("decode request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if len(payload) != 1 || payload["query"] != expectedQuery {
			t.Errorf("unexpected GraphQL payload: %#v", payload)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, response)
	}))
	defer server.Close()

	exporter := Exporter{URI: server.URL}
	body, err := exporter.fetch()
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if string(body) != response {
		t.Errorf("response = %q, want %q", body, response)
	}
}
