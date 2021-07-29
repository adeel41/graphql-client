package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"golang.org/x/net/context/ctxhttp"
)

// Client is a client for interacting with a GraphQL API.
type Client struct {
	url        string
	httpClient *http.Client
}

// NewClient makes a new Client capable of making GraphQL requests.
func NewClient(url string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{
		url:        url,
		httpClient: httpClient,
	}
}

func (c *Client) RawRequest(ctx context.Context, query string, variables map[string]interface{}) (*HttpResponse, error) {
	serverErrorResponse := &HttpResponse{
		StatusCode: 500,
	}
	requestData := struct {
		Query     string                 `json:"query"`
		Variables map[string]interface{} `json:"variables,omitempty"`
	}{
		Query:     query,
		Variables: variables,
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	err := encoder.Encode(requestData)
	if err != nil {
		return serverErrorResponse, err
	}

	resp, err := ctxhttp.Post(ctx, c.httpClient, c.url, "application/json", &buf)
	if err != nil {
		return serverErrorResponse, err
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	serverErrorResponse.StatusCode = resp.StatusCode
	if resp.StatusCode != http.StatusOK {
		return serverErrorResponse, fmt.Errorf("non-200 OK status code: %v body: %q", resp.Status, body)
	}

	return &HttpResponse{
		StatusCode: resp.StatusCode,
		Data:       body,
	}, nil
}

type HttpResponse struct {
	StatusCode int
	Data       []byte
}
