package graphql_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/adeel41/graphql-client"
)

func TestClient_JsonEncodingError_ReturnsError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, req *http.Request) {
		mustWrite(w, "{{invalid jon}}")
	})

	client := graphql.NewClient("/graphql", &http.Client{
		Transport: localRoundTripper{handler: mux},
	})
	resp, err := client.RawRequest(context.Background(), "", nil)
	if err == nil {
		t.Error("An error should have returned because of invalid json")
	}

	if resp != nil {
		t.Error("Returned response shoould have been nil because of invalid json")
	}
}

func TestClient_GraphQLServerError_ReturnsError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, req *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	})

	client := graphql.NewClient("/graphql", &http.Client{
		Transport: localRoundTripper{handler: mux},
	})
	resp, err := client.RawRequest(context.Background(), "", nil)
	if err == nil {
		t.Error("An error should have returned when GraphQL server returned an error")
	}

	if resp == nil {
		t.Error("Response object is nil")
		t.FailNow()
	}

	if resp.StatusCode != 500 {
		t.Errorf("Should have set the status code. Expected %d but received %d", 500, resp.StatusCode)
	}
}

type localRoundTripper struct {
	handler http.Handler
}

func (l localRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	l.handler.ServeHTTP(w, req)
	return w.Result(), nil
}

func mustWrite(w io.Writer, s string) {
	_, err := io.WriteString(w, s)
	if err != nil {
		panic(err)
	}
}
