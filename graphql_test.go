package graphql_test

import (
	"context"
	"io"
	stdlog "log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/adeel41/graphql-client"
	"github.com/go-logr/stdr"
)

func createClient(handler func(http.ResponseWriter, *http.Request)) *graphql.Client {
	stdr.SetVerbosity(0)
	log := stdr.NewWithOptions(stdlog.New(os.Stderr, "", stdlog.LstdFlags), stdr.Options{LogCaller: stdr.All})
	log = log.WithName("MyName")

	mux := http.NewServeMux()
	mux.HandleFunc("/graphql", handler)
	return graphql.NewClient("/graphql", &http.Client{
		Transport: localRoundTripper{handler: mux},
	}, &log)
}

func TestClient_JsonEncodingError_ReturnsError(t *testing.T) {
	client := createClient(func(w http.ResponseWriter, req *http.Request) {
		mustWrite(w, "{{invalid jon}}")
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
	client := createClient(func(w http.ResponseWriter, req *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
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

func TestClient_GraphQLServerReturnsData_ReturnsSuccess(t *testing.T) {
	expectedResponse := string(`{"data": { "me": { "id": "123456"} } }`)
	client := createClient(func(w http.ResponseWriter, req *http.Request) {
		mustWrite(w, expectedResponse)
	})

	resp, err := client.RawRequest(context.Background(), "query { me { id } }", nil)
	if err != nil {
		t.Error("Should have returned a nil error")
	}
	if resp.StatusCode != 200 {
		t.Error("Should have returned a success error code")
	}

	content := string(resp.ResponseContent)
	if content != expectedResponse {
		t.Errorf("Expected %s but received %s", expectedResponse, content)
	}
	t.Log(content)
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
