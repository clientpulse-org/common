package mocks

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/quiby-ai/common/pkg/httpx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestClientMockUsage(t *testing.T) {
	mockClient := NewClient(t)

	expectedResp := httpx.Response{
		Status: 200,
		Body:   []byte("mocked response"),
		Headers: map[string][]string{
			"Content-Type": {"application/json"},
		},
		URL: "https://example.com",
	}

	mockClient.On("Do", mock.Anything, mock.Anything).Return(expectedResp, nil)

	resp, err := mockClient.Do(context.Background(), httpx.Request{
		Method: "GET",
		URL:    "https://example.com",
	})

	assert.NoError(t, err)
	assert.Equal(t, expectedResp, resp)
	mockClient.AssertExpectations(t)
}

func TestClientMockWithError(t *testing.T) {
	mockClient := NewClient(t)
	expectedErr := errors.New("network error")

	mockClient.On("Do", mock.Anything, mock.Anything).Return(httpx.Response{}, expectedErr)

	resp, err := mockClient.Do(context.Background(), httpx.Request{
		Method: "GET",
		URL:    "https://example.com",
	})

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, httpx.Response{}, resp)
	mockClient.AssertExpectations(t)
}

func TestClientMockDoGET(t *testing.T) {
	mockClient := NewClient(t)
	expectedResp := httpx.Response{
		Status: 200,
		Body:   []byte("GET response"),
		URL:    "https://example.com",
	}

	mockClient.On("DoGET", mock.Anything, "https://example.com", mock.Anything, mock.Anything).Return(expectedResp, nil)

	resp, err := mockClient.DoGET(context.Background(), "https://example.com", nil, nil)

	assert.NoError(t, err)
	assert.Equal(t, expectedResp, resp)
	mockClient.AssertExpectations(t)
}

func TestClientMockWithSpecificRequest(t *testing.T) {
	mockClient := NewClient(t)
	expectedResp := httpx.Response{
		Status: 201,
		Body:   []byte("created"),
		URL:    "https://api.example.com/users",
	}

	expectedReq := httpx.Request{
		Method: "POST",
		URL:    "https://api.example.com/users",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: nil,
	}

	mockClient.On("Do", mock.Anything, expectedReq).Return(expectedResp, nil)

	resp, err := mockClient.Do(context.Background(), expectedReq)

	assert.NoError(t, err)
	assert.Equal(t, expectedResp, resp)
	mockClient.AssertExpectations(t)
}

func TestClientMockWithContext(t *testing.T) {
	mockClient := NewClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	expectedResp := httpx.Response{
		Status: 200,
		Body:   []byte("timeout response"),
		URL:    "https://example.com",
	}

	mockClient.On("Do", ctx, mock.Anything).Return(expectedResp, nil)

	resp, err := mockClient.Do(ctx, httpx.Request{
		Method: "GET",
		URL:    "https://example.com",
	})

	assert.NoError(t, err)
	assert.Equal(t, expectedResp, resp)
	mockClient.AssertExpectations(t)
}

func TestClientMockMultipleCalls(t *testing.T) {
	mockClient := NewClient(t)

	call1 := mockClient.On("Do", mock.Anything, httpx.Request{URL: "https://api1.com"}).Return(httpx.Response{Status: 200, URL: "https://api1.com"}, nil)
	call2 := mockClient.On("Do", mock.Anything, httpx.Request{URL: "https://api2.com"}).Return(httpx.Response{Status: 404, URL: "https://api2.com"}, nil)

	resp1, err1 := mockClient.Do(context.Background(), httpx.Request{URL: "https://api1.com"})
	resp2, err2 := mockClient.Do(context.Background(), httpx.Request{URL: "https://api2.com"})

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, 200, resp1.Status)
	assert.Equal(t, 404, resp2.Status)

	call1.Unset()
	call2.Unset()
	mockClient.AssertExpectations(t)
}
