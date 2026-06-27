package hrobot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const DefaultEndpoint = "https://robot-ws.your-server.de"

type Client struct {
	Endpoint   string
	Username   string
	Password   string
	HTTPClient *http.Client
}

func NewClient(endpoint, username, password string) *Client {
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	return &Client{
		Endpoint:   strings.TrimRight(endpoint, "/"),
		Username:   username,
		Password:   password,
		HTTPClient: &http.Client{},
	}
}

type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("hetzner robot API error (HTTP %d): %s: %s", e.Status, e.Code, e.Message)
	}
	return fmt.Sprintf("hetzner robot API error (HTTP %d): %s", e.Status, e.Message)
}

func IsNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound
}

type errorEnvelope struct {
	Error struct {
		Status  int    `json:"status"`
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c *Client) do(ctx context.Context, method, path string, form url.Values) ([]byte, error) {
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, method, c.Endpoint+path, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.SetBasicAuth(c.Username, c.Password)
	req.Header.Set("Accept", "application/json")
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := &APIError{Status: resp.StatusCode, Message: strings.TrimSpace(string(respBytes))}
		var env errorEnvelope
		if json.Unmarshal(respBytes, &env) == nil && env.Error.Code != "" {
			apiErr.Code = env.Error.Code
			apiErr.Message = env.Error.Message
		}
		return nil, apiErr
	}

	return respBytes, nil
}

type Subnet struct {
	IP   string `json:"ip"`
	Mask string `json:"mask"`
}

type Server struct {
	ServerNumber  int      `json:"server_number"`
	ServerIP      string   `json:"server_ip"`
	ServerIPv6Net string   `json:"server_ipv6_net"`
	ServerName    string   `json:"server_name"`
	Product       string   `json:"product"`
	DC            string   `json:"dc"`
	Traffic       string   `json:"traffic"`
	Status        string   `json:"status"`
	Cancelled     bool     `json:"cancelled"`
	PaidUntil     string   `json:"paid_until"`
	IPs           []string `json:"ip"`
	Subnets       []Subnet `json:"subnet"`
}

type serverEnvelope struct {
	Server Server `json:"server"`
}

func (c *Client) GetServer(ctx context.Context, number int) (*Server, error) {
	body, err := c.do(ctx, http.MethodGet, "/server/"+strconv.Itoa(number), nil)
	if err != nil {
		return nil, err
	}

	var env serverEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("decoding server response: %w", err)
	}
	return &env.Server, nil
}

func (c *Client) ListServers(ctx context.Context) ([]Server, error) {
	body, err := c.do(ctx, http.MethodGet, "/server", nil)
	if err != nil {
		return nil, err
	}

	var envs []serverEnvelope
	if err := json.Unmarshal(body, &envs); err != nil {
		return nil, fmt.Errorf("decoding server list response: %w", err)
	}

	servers := make([]Server, len(envs))
	for i := range envs {
		servers[i] = envs[i].Server
	}
	return servers, nil
}

func (c *Client) ServerByIP(ctx context.Context, ip string) (*Server, error) {
	servers, err := c.ListServers(ctx)
	if err != nil {
		return nil, err
	}

	for i := range servers {
		if servers[i].ServerIP == ip {
			return &servers[i], nil
		}
		for _, addr := range servers[i].IPs {
			if addr == ip {
				return &servers[i], nil
			}
		}
	}

	return nil, &APIError{
		Status:  http.StatusNotFound,
		Code:    "SERVER_NOT_FOUND",
		Message: fmt.Sprintf("no server found with IP %s", ip),
	}
}

type Key struct {
	Name        string `json:"name"`
	Fingerprint string `json:"fingerprint"`
	Type        string `json:"type"`
	Size        int    `json:"size"`
	Data        string `json:"data"`
	CreatedAt   string `json:"created_at"`
}

type keyEnvelope struct {
	Key Key `json:"key"`
}

func (c *Client) GetKey(ctx context.Context, fingerprint string) (*Key, error) {
	body, err := c.do(ctx, http.MethodGet, "/key/"+fingerprint, nil)
	if err != nil {
		return nil, err
	}

	var env keyEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("decoding key response: %w", err)
	}
	return &env.Key, nil
}

func (c *Client) CreateKey(ctx context.Context, name, data string) (*Key, error) {
	body, err := c.do(ctx, http.MethodPost, "/key", url.Values{
		"name": {name},
		"data": {data},
	})
	if err != nil {
		return nil, err
	}

	var env keyEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("decoding key response: %w", err)
	}
	return &env.Key, nil
}

func (c *Client) UpdateKey(ctx context.Context, fingerprint, name string) (*Key, error) {
	body, err := c.do(ctx, http.MethodPost, "/key/"+fingerprint, url.Values{
		"name": {name},
	})
	if err != nil {
		return nil, err
	}

	var env keyEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("decoding key response: %w", err)
	}
	return &env.Key, nil
}

func (c *Client) DeleteKey(ctx context.Context, fingerprint string) error {
	_, err := c.do(ctx, http.MethodDelete, "/key/"+fingerprint, nil)
	return err
}
