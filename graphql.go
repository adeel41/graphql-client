package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"golang.org/x/net/context/ctxhttp"
)

var logger *log.Logger

func init() {
	logger = log.New(os.Stderr, "graphql-client", log.Ldate|log.Ltime|log.Lshortfile)
}

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

	logger.Printf("[POST] %v\r\n%q\r\n", c.url, buf.Bytes())
	responseContent, err := ctxhttp.Post(ctx, c.httpClient, c.url, "application/json", &buf)
	var errBody []byte
	if err != nil {
		errBody, _ = ioutil.ReadAll(responseContent.Body)
		logger.Printf("%v status code. %q\r\n%s\r\n", responseContent.Status, err.Error(), errBody)
		return &GraphQLResponse{
			StatusCode: responseContent.StatusCode,
		}, err
	}
	defer responseContent.Body.Close()

	var body []byte
	if errBody != nil {
		body = errBody
	} else {
		body, _ = ioutil.ReadAll(responseContent.Body)
	}

	if responseContent.StatusCode != http.StatusOK {
		errString := fmt.Sprintf("%v status code\r\n%s", responseContent.Status, body)
		logger.Println(errString)
		return &GraphQLResponse{
			StatusCode:      responseContent.StatusCode,
			ResponseContent: body,
		}, fmt.Errorf(errString)
	}

	result, err := decodeGraphQLResponse(body)
	if err != nil {
		logger.Printf("response decoding error: %s\r\n", err.Error())
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
