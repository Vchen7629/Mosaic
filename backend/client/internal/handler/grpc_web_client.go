package handler

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type RetryConfig struct {
	MaxAttempts       int
	InitialBackoff    time.Duration
	MaxBackoff        time.Duration
	BackoffMultiplier float64
	RetryableCodes    []codes.Code
}

var DefaultRetryConfig = RetryConfig{
	MaxAttempts:       4,
	InitialBackoff:    time.Second,
	MaxBackoff:        10 * time.Second,
	BackoffMultiplier: 2.0,
	RetryableCodes:    []codes.Code{codes.Unavailable},
}

var NoRetry = RetryConfig{MaxAttempts: 1}

type Client struct {
	baseURL    string
	httpClient *http.Client
	retry      RetryConfig
}

func NewClient(baseURL string, retry RetryConfig) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{},
		retry:      retry,
	}
}

func (c *Client) Invoke(ctx context.Context, method string, args, reply interface{}, opts ...grpc.CallOption) error {
	backoff := c.retry.InitialBackoff
	var lastErr error

	for attempt := 0; attempt < c.retry.MaxAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return status.FromContextError(ctx.Err()).Err()
			case <-time.After(backoff):
			}
			backoff = time.Duration(float64(backoff) * c.retry.BackoffMultiplier)
			if backoff > c.retry.MaxBackoff {
				backoff = c.retry.MaxBackoff
			}
		}

		lastErr = c.invoke(ctx, method, args, reply)
		if lastErr == nil {
			return nil
		}
		if !c.isRetryable(lastErr) {
			return lastErr
		}
	}
	return lastErr
}

func (c *Client) isRetryable(err error) bool {
	s, ok := status.FromError(err)
	if !ok {
		return false
	}
	for _, code := range c.retry.RetryableCodes {
		if s.Code() == code {
			return true
		}
	}
	return false
}

func (c *Client) invoke(ctx context.Context, method string, args, reply interface{}) error {
	reqMsg, ok := args.(proto.Message)
	if !ok {
		return status.Error(codes.Internal, "args must be proto.Message")
	}
	reqBytes, err := proto.Marshal(reqMsg)
	if err != nil {
		return status.Errorf(codes.Internal, "marshal request: %v", err)
	}

	frame := make([]byte, 5+len(reqBytes))
	frame[0] = 0
	binary.BigEndian.PutUint32(frame[1:5], uint32(len(reqBytes)))
	copy(frame[5:], reqBytes)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+method, bytes.NewReader(frame))
	if err != nil {
		return status.Errorf(codes.Internal, "create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/grpc-web+proto")
	req.Header.Set("X-Grpc-Web", "1")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return status.Errorf(codes.Unavailable, "http request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return status.Errorf(codes.Internal, "http status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return status.Errorf(codes.Internal, "read body: %v", err)
	}

	var grpcStatus codes.Code
	var grpcMessage string
	dataRead := false
	pos := 0

	for pos < len(body) {
		if pos+5 > len(body) {
			return status.Error(codes.Internal, "truncated frame header")
		}
		frameType := body[pos]
		frameLen := int(binary.BigEndian.Uint32(body[pos+1 : pos+5]))
		pos += 5

		if pos+frameLen > len(body) {
			return status.Error(codes.Internal, "truncated frame body")
		}
		frameData := body[pos : pos+frameLen]
		pos += frameLen

		switch frameType {
		case 0x00:
			replyMsg, ok := reply.(proto.Message)
			if !ok {
				return status.Error(codes.Internal, "reply must be proto.Message")
			}
			if err := proto.Unmarshal(frameData, replyMsg); err != nil {
				return status.Errorf(codes.Internal, "unmarshal response: %v", err)
			}
			dataRead = true
		case 0x80:
			for _, line := range strings.Split(string(frameData), "\r\n") {
				k, v, ok := strings.Cut(line, ":")
				if !ok {
					continue
				}
				switch strings.TrimSpace(k) {
				case "grpc-status":
					var code int
					fmt.Sscanf(strings.TrimSpace(v), "%d", &code)
					grpcStatus = codes.Code(code)
				case "grpc-message":
					grpcMessage, _ = url.PathUnescape(strings.TrimSpace(v))
				}
			}
		}
	}

	if !dataRead && grpcStatus == codes.OK {
		return status.Error(codes.Internal, "no data frame in response")
	}
	if grpcStatus != codes.OK {
		return status.Error(grpcStatus, grpcMessage)
	}
	return nil
}

func (c *Client) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, status.Error(codes.Unimplemented, "streaming not supported")
}

func (c *Client) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}
