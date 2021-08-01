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

func (c *Client) RawRequest(ctx context.Context, query string, variables map[string]interface{}) (*GraphQLResponse, error) {
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
		return nil, err
	}

	responseContent, err := ctxhttp.Post(ctx, c.httpClient, c.url, "application/json", &buf)
	if err != nil {
		return &GraphQLResponse{
			StatusCode: responseContent.StatusCode,
		}, err
	}
	defer responseContent.Body.Close()

	body, _ := ioutil.ReadAll(responseContent.Body)
	if responseContent.StatusCode != http.StatusOK {
		return &GraphQLResponse{
			StatusCode:      responseContent.StatusCode,
			ResponseContent: body,
		}, fmt.Errorf("non-200 OK status code: %v body: %q", responseContent.Status, body)
	}

	result, err := decodeGraphQLResponse(body)
	if err != nil {
		return nil, err
	}

	result.StatusCode = responseContent.StatusCode
	result.ResponseContent = body
	return result, nil
}

func decodeGraphQLResponse(data []byte) (*GraphQLResponse, error) {
	reader := bytes.NewReader(data)
	decoder := json.NewDecoder(reader)
	resp := &GraphQLResponse{}
	err := decoder.Decode(resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

type GraphQLResponse struct {
	StatusCode      int
	ResponseContent []byte
	Data            *json.RawMessage
	Errors          GraphQLError
	Extensions      *json.RawMessage
}

type GraphQLError []struct {
	Message   string
	Locations []struct {
		Line   int
		Column int
	}
}
