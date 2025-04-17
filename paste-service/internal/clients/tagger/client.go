package tagger

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

var (
	ErrTaggerUnavailable = errors.New("сервис тегирования недоступен")
	ErrInvalidResponse   = errors.New("некорректный ответ от сервиса тегирования")
)

type TaggerClient interface {
	GetTags(text string) ([]string, error)
}

type Config struct {
	BaseURL     string
	Timeout     time.Duration
	MaxTextSize int
}

func DefaultConfig() Config {
	return Config{
		BaseURL:     "http://tagger-ml:8000",
		Timeout:     5 * time.Second,
		MaxTextSize: 10000, // 10KB
	}
}

type HTTPClient struct {
	client      *http.Client
	baseURL     string
	maxTextSize int
}

type tagRequest struct {
	Text string `json:"text"`
}

type tagResponse struct {
	Tags []string `json:"tags"`
}

func NewHTTPClient(cfg Config) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		baseURL:     cfg.BaseURL,
		maxTextSize: cfg.MaxTextSize,
	}
}

func (c *HTTPClient) GetTags(text string) ([]string, error) {
	if len(text) > c.maxTextSize {
		text = text[:c.maxTextSize]
	}

	reqBody := tagRequest{Text: text}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Post(
		fmt.Sprintf("%s/api/tags", c.baseURL),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, ErrTaggerUnavailable
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: код ответа %d", ErrTaggerUnavailable, resp.StatusCode)
	}

	var response tagResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, ErrInvalidResponse
	}

	return response.Tags, nil
}

type MockClient struct {
	tags []string
	err  error
}

func NewMockClient(tags []string, err error) *MockClient {
	return &MockClient{
		tags: tags,
		err:  err,
	}
}

func (c *MockClient) GetTags(text string) ([]string, error) {
	if c.err != nil {
		return nil, c.err
	}
	return c.tags, nil
}
