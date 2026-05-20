// Package cgrates provides a JSON-RPC client for CGRateS integration.
package cgrates

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client handles communication with CGRateS
type Client struct {
	baseURL string
	client  *http.Client
}

// NewClient creates a new CGRateS client
func NewClient(host string, port int, timeout time.Duration) *Client {
	return &Client{
		baseURL: fmt.Sprintf("http://%s:%d/jsonrpc", host, port),
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// RPCRequest represents a JSON-RPC request
type RPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      int         `json:"id"`
}

// RPCResponse represents a JSON-RPC response
type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
	ID      int             `json:"id"`
}

// RPCError represents a JSON-RPC error
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("cgrates rpc error %d: %s", e.Code, e.Message)
}

// Call executes a JSON-RPC method
func (c *Client) Call(ctx context.Context, method string, params interface{}, result interface{}) error {
	reqBody := RPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var rpcResp RPCResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}

	if rpcResp.Error != nil {
		return rpcResp.Error
	}

	if result != nil && rpcResp.Result != nil {
		if err := json.Unmarshal(rpcResp.Result, result); err != nil {
			return fmt.Errorf("unmarshal result: %w", err)
		}
	}

	return nil
}

// Common CGRateS methods wrapper

// GetAccount retrieves account information
func (c *Client) GetAccount(ctx context.Context, tenant, account string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := c.Call(ctx, "ApierV1.GetAccount", map[string]string{
		"Tenant":  tenant,
		"Account": account,
	}, &result)
	return result, err
}

// AddBalance adds balance to an account
func (c *Client) AddBalance(ctx context.Context, tenant, account, balanceType string, value float64, directions string) error {
	return c.Call(ctx, "ApierV1.AddBalance", map[string]interface{}{
		"Tenant":      tenant,
		"Account":     account,
		"BalanceType": balanceType,
		"Value":       value,
		"Directions":  directions,
	}, nil)
}

// GetCDRs retrieves CDR records
func (c *Client) GetCDRs(ctx context.Context, filter map[string]interface{}) ([]map[string]interface{}, error) {
	var result []map[string]interface{}
	err := c.Call(ctx, "ApierV1.GetCDRs", filter, &result)
	return result, err
}

// GetActiveSessions returns active sessions
func (c *Client) GetActiveSessions(ctx context.Context, filter map[string]interface{}) ([]map[string]interface{}, error) {
	var result []map[string]interface{}
	err := c.Call(ctx, "SMGv1.GetActiveSessions", filter, &result)
	return result, err
}
