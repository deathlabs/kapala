// Package embeddings creates vector embeddings with an OpenAI-compatible API.
package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	defaultEndpoint = "https://api.openai.com/v1/embeddings"
	defaultModel    = "text-embedding-3-small"
)

// Client sends embedding requests to an OpenAI-compatible API.
type Client struct {
	apiKey     string
	endpoint   string
	httpClient *http.Client
	model      string
}

// Option configures a Client.
type Option func(*Client) error

type apiErrorResponse struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Message string `json:"message"`
}

type embeddingDatum struct {
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
}

type embeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

type embeddingResponse struct {
	Data []embeddingDatum `json:"data"`
}

// Model returns the model configured for this client.
func (client *Client) Model() string {
	return client.model
}

// NewClient creates an embeddings client.
func NewClient(apiKey string, options ...Option) (*Client, error) {
	var (
		client *Client
		err    error
		option Option
	)

	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("embedding API key is empty")
	}

	client = &Client{
		apiKey:     apiKey,
		endpoint:   defaultEndpoint,
		httpClient: http.DefaultClient,
		model:      defaultModel,
	}

	for _, option = range options {
		if option == nil {
			return nil, errors.New("embedding client option is nil")
		}

		err = option(client)
		if err != nil {
			return nil, err
		}
	}

	return client, nil
}

// WithEndpoint configures the embeddings API endpoint.
func WithEndpoint(endpoint string) Option {
	return func(client *Client) error {
		if strings.TrimSpace(endpoint) == "" {
			return errors.New("embedding endpoint is empty")
		}

		client.endpoint = endpoint

		return nil
	}
}

// WithHTTPClient configures the HTTP client used for requests.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(client *Client) error {
		if httpClient == nil {
			return errors.New("embedding HTTP client is nil")
		}

		client.httpClient = httpClient

		return nil
	}
}

// WithModel configures the embeddings model.
func WithModel(model string) Option {
	return func(client *Client) error {
		if strings.TrimSpace(model) == "" {
			return errors.New("embedding model is empty")
		}

		client.model = model

		return nil
	}
}

// Embed returns one vector for each input, preserving input order.
func (client *Client) Embed(ctx context.Context, inputs []string) ([][]float64, error) {
	var (
		body       []byte
		embeddings [][]float64
		err        error
		request    *http.Request
		response   *http.Response
	)

	err = validateInputs(inputs)
	if err != nil {
		return nil, err
	}

	body, err = json.Marshal(embeddingRequest{Input: inputs, Model: client.model})
	if err != nil {
		return nil, err
	}

	request, err = http.NewRequestWithContext(ctx, http.MethodPost, client.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Authorization", "Bearer "+client.apiKey)
	request.Header.Set("Content-Type", "application/json")

	response, err = client.httpClient.Do(request)
	if err != nil {
		return nil, err
	}

	body, err = readAndClose(response.Body)
	if err != nil {
		return nil, err
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, responseError(response.StatusCode, body)
	}

	embeddings, err = decodeEmbeddings(body, len(inputs))
	if err != nil {
		return nil, err
	}

	return embeddings, nil
}

func decodeEmbeddings(body []byte, inputCount int) ([][]float64, error) {
	var (
		datum      embeddingDatum
		embeddings [][]float64
		err        error
		response   embeddingResponse
		seen       []bool
	)

	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	if len(response.Data) != inputCount {
		return nil, fmt.Errorf("embedding response contains %d vectors for %d inputs", len(response.Data), inputCount)
	}

	embeddings = make([][]float64, inputCount)
	seen = make([]bool, inputCount)
	for _, datum = range response.Data {
		if datum.Index < 0 || datum.Index >= inputCount {
			return nil, fmt.Errorf("embedding response index %d is out of range", datum.Index)
		}
		if seen[datum.Index] {
			return nil, fmt.Errorf("embedding response index %d is duplicated", datum.Index)
		}
		if len(datum.Embedding) == 0 {
			return nil, fmt.Errorf("embedding response vector %d is empty", datum.Index)
		}

		embeddings[datum.Index] = datum.Embedding
		seen[datum.Index] = true
	}

	return embeddings, nil
}

func readAndClose(body io.ReadCloser) ([]byte, error) {
	var (
		contents []byte
		err      error
	)

	contents, err = io.ReadAll(body)
	if err != nil {
		_ = body.Close()
		return nil, err
	}

	err = body.Close()
	if err != nil {
		return nil, err
	}

	return contents, nil
}

func responseError(statusCode int, body []byte) error {
	var (
		err      error
		response apiErrorResponse
	)

	err = json.Unmarshal(body, &response)
	if err == nil && strings.TrimSpace(response.Error.Message) != "" {
		return fmt.Errorf("embedding request failed with status %d: %s", statusCode, response.Error.Message)
	}

	return fmt.Errorf("embedding request failed with status %d", statusCode)
}

func validateInputs(inputs []string) error {
	var (
		index int
		input string
	)

	if len(inputs) == 0 {
		return errors.New("embedding inputs are empty")
	}

	for index, input = range inputs {
		if strings.TrimSpace(input) == "" {
			return fmt.Errorf("embedding input at index %d is empty", index)
		}
	}

	return nil
}
