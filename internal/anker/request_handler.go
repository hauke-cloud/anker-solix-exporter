package anker

import (
	"encoding/json"
	"fmt"
	"io"

	"go.uber.org/zap"
)

// APIResponse represents the common structure of all Anker API responses
type APIResponse interface {
	GetCode() int
	GetMsg() string
}

// BaseResponse contains the common fields in all API responses
type BaseResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (r *BaseResponse) GetCode() int {
	return r.Code
}

func (r *BaseResponse) GetMsg() string {
	return r.Msg
}

// RequestHandler handles the common request/response flow for API calls
type RequestHandler struct {
	client *Client
}

// newRequestHandler creates a new request handler
func newRequestHandler(client *Client) *RequestHandler {
	return &RequestHandler{client: client}
}

// execute performs a complete API request with JSON marshaling/unmarshaling
func (h *RequestHandler) execute(
	method string,
	endpoint string,
	request interface{},
	response APIResponse,
	needAuth bool,
) error {
	// Marshal request body if provided
	var body []byte
	var err error
	if request != nil {
		body, err = json.Marshal(request)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
	}

	// Perform HTTP request
	resp, err := h.client.doRequest(method, endpoint, body, needAuth)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	h.client.debugLog("API response received",
		zap.String("endpoint", endpoint),
		zap.Int("response_size", len(bodyBytes)),
	)

	// Unmarshal response
	if err := json.Unmarshal(bodyBytes, response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Check response code
	if response.GetCode() != 0 {
		return fmt.Errorf("API error: %s (code: %d)", response.GetMsg(), response.GetCode())
	}

	return nil
}

// executeRaw performs an API request and returns the raw http.Response for custom handling
// func (h *RequestHandler) executeRaw(
// 	method string,
// 	endpoint string,
// 	request interface{},
// 	needAuth bool,
// ) (*http.Response, error) {
// 	// Marshal request body if provided
// 	var body []byte
// 	var err error
// 	if request != nil {
// 		body, err = json.Marshal(request)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to marshal request: %w", err)
// 		}
// 	}
//
// 	// Perform HTTP request
// 	resp, err := h.client.doRequest(method, endpoint, body, needAuth)
// 	if err != nil {
// 		return nil, fmt.Errorf("request failed: %w", err)
// 	}
//
// 	return resp, nil
// }
