package httpapi

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestHealthEndpointReturnsOK(t *testing.T) {
	var logs bytes.Buffer
	router := NewRouter(log.New(&logs, "", 0))

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	if diff := cmp.Diff(`{"status":"ok"}`, response.Body.String()); diff != "" {
		t.Fatalf("body mismatch (-want +got):\n%s", diff)
	}

	if !bytes.Contains(logs.Bytes(), []byte("health endpoint served")) {
		t.Fatalf("log output = %q, want health trace", logs.String())
	}
}
