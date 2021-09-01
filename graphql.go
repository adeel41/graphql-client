package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/go-logr/logr"
	"golang.org/x/net/context/ctxhttp"
)

// Client is a client for interacting with a GraphQL API.
type Client struct {
	url        string
	httpClient *http.Client
	logger     *logr.Logger
}

// NewClient makes a new Client capable of making GraphQL requests.
func NewClient(url string, httpClient *http.Client, logger *logr.Logger) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{
		url:        url,
		httpClient: httpClient,
		logger:     logger,
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

	logger := c.logger

	logger.Info("[POST] graphql request", "url", c.url)
	logger.V(1).Info("data sent in request", "data", fmt.Sprintf("%q", buf.Bytes()))
	responseContent, err := ctxhttp.Post(ctx, c.httpClient, c.url, "application/json", &buf)

	var errBody []byte
	if err != nil {
		if responseContent != nil {
			errBody, _ = ioutil.ReadAll(responseContent.Body)
			logger.Error(err, "error returned from graphql request", "status", responseContent.Status)
			logger.V(1).Info("response data received", "body", string(errBody))
			return &GraphQLResponse{
				StatusCode: responseContent.StatusCode,
			}, err
		} else {
			logger.Error(err, "Error in graphql request")
			logger.V(1).Info("Request body", "data", buf.String())
			return nil, err
		}
	}
	defer responseContent.Body.Close()

	var body []byte
	if errBody != nil {
		body = errBody
	} else {
		body, _ = ioutil.ReadAll(responseContent.Body)
	}

	if responseContent.StatusCode != http.StatusOK {

		err := fmt.Errorf("%v", responseContent.Status)
		logger.Error(err, "graphql returned a non-200 response", "status code", responseContent.StatusCode)
		logger.V(1).Info("response data received", "body", string(body))
		return &GraphQLResponse{
			StatusCode:      responseContent.StatusCode,
			ResponseContent: body,
		}, err
	}

	result, err := decodeGraphQLResponse(body)
	if err != nil {
		logger.Error(err, "error while decoding graphql response")
		logger.V(1).Info("json data", "data", string(body))
		return nil, err
	}

	result.StatusCode = responseContent.StatusCode
	result.ResponseContent = body
	logger.Info("success", "status code", responseContent.StatusCode)
	logger.V(1).Info("response content", "content", string(body))
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
