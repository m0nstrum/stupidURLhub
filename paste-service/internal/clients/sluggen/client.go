package sluggen

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	ErrSlugGeneratorUnavailable = errors.New("сервис генерации slug недоступен")
	ErrInvalidResponse          = errors.New("некорректный ответ от сервиса генерации slug")
)

type SlugClient interface {
	GenerateSlug(content string, tags []string) (string, error)
}

type Config struct {
	Address     string
	Timeout     time.Duration
	MaxTextSize int
}

func DefaultConfig() Config {
	return Config{
		Address:     "slug-generator:50051",
		Timeout:     5 * time.Second,
		MaxTextSize: 10000, // 10KB
	}
}

type GRPCClient struct {
	conn        *grpc.ClientConn
	client      SlugGeneratorClient
	timeout     time.Duration
	maxTextSize int
}

func NewGRPCClient(cfg Config) (*GRPCClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		cfg.Address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, ErrSlugGeneratorUnavailable
	}

	return &GRPCClient{
		conn:        conn,
		client:      NewSlugGeneratorClient(conn),
		timeout:     cfg.Timeout,
		maxTextSize: cfg.MaxTextSize,
	}, nil
}

func (c *GRPCClient) Close() error {
	return c.conn.Close()
}

func (c *GRPCClient) GenerateSlug(content string, tags []string) (string, error) {
	if len(content) > c.maxTextSize {
		content = content[:c.maxTextSize]
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.client.GenerateSlug(ctx, &GenerateSlugRequest{
		Content: content,
		Tags:    tags,
	})
	if err != nil {
		return "", ErrSlugGeneratorUnavailable
	}

	if resp.Slug == "" {
		return "", ErrInvalidResponse
	}

	return resp.Slug, nil
}

type MockClient struct {
	slug string
	err  error
}

func NewMockClient(slug string, err error) *MockClient {
	return &MockClient{
		slug: slug,
		err:  err,
	}
}

func (c *MockClient) GenerateSlug(content string, tags []string) (string, error) {
	if c.err != nil {
		return "", c.err
	}
	return c.slug, nil
}

type SlugGeneratorClient interface {
	GenerateSlug(ctx context.Context, in *GenerateSlugRequest, opts ...grpc.CallOption) (*GenerateSlugResponse, error)
}

type GenerateSlugRequest struct {
	Content string   `protobuf:"bytes,1,opt,name=content,proto3" json:"content,omitempty"`
	Tags    []string `protobuf:"bytes,2,rep,name=tags,proto3" json:"tags,omitempty"`
}

type GenerateSlugResponse struct {
	Slug string `protobuf:"bytes,1,opt,name=slug,proto3" json:"slug,omitempty"`
}

func NewSlugGeneratorClient(cc *grpc.ClientConn) SlugGeneratorClient {
	return &mockSlugGeneratorClient{cc}
}

type mockSlugGeneratorClient struct {
	cc *grpc.ClientConn
}

func (c *mockSlugGeneratorClient) GenerateSlug(ctx context.Context, in *GenerateSlugRequest, opts ...grpc.CallOption) (*GenerateSlugResponse, error) {
	// пока что фиктивный
	return &GenerateSlugResponse{Slug: "generated-slug-for-content"}, nil
}
