package portainer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type PortainerSource struct {
	PortainerURL string `json:"portainer_url"`
	StackID      int    `json:"stack_id"`
	EndpointID   int    `json:"endpoint_id"`
	APIKey       string `json:"api_key,omitempty"`
	HasAPIKey    bool   `json:"has_api_key,omitempty"`
}

type Endpoint struct {
	ID   int    `json:"Id"`
	Name string `json:"Name"`
}

type Stack struct {
	ID           int    `json:"Id"`
	Name         string `json:"Name"`
	EndpointID   int    `json:"EndpointId"`
	Status       int    `json:"Status"`
	IsDvbBacked bool   `json:"is_dvb_backed"`
	EndpointName string `json:"endpoint_name,omitempty"`
}

type StackFile struct {
	Content string `json:"StackFileContent"`
}

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

func (c *Client) do(ctx context.Context, method, apiPath string, body io.Reader) (*http.Response, error) {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base URL: %w", err)
	}
	ref, err := url.Parse(apiPath)
	if err != nil {
		return nil, fmt.Errorf("parse API path: %w", err)
	}
	target := base.ResolveReference(ref)
	if target.Host != base.Host {
		return nil, fmt.Errorf("request host %q does not match base %q", target.Host, base.Host)
	}
	req, err := http.NewRequestWithContext(ctx, method, target.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", c.apiKey)
	return c.httpClient.Do(req)
}

func (c *Client) TestConnection(ctx context.Context) error {
	resp, err := c.do(ctx, "GET", "/api/system/status", nil)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("portainer returned status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) ListEndpoints(ctx context.Context) ([]Endpoint, error) {
	resp, err := c.do(ctx, "GET", "/api/endpoints", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var endpoints []Endpoint
	if err := json.NewDecoder(resp.Body).Decode(&endpoints); err != nil {
		return nil, err
	}
	return endpoints, nil
}

func (c *Client) ListStacks(ctx context.Context, endpointID int) ([]Stack, error) {
	resp, err := c.do(ctx, "GET", "/api/stacks", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var allStacks []Stack
	if err := json.NewDecoder(resp.Body).Decode(&allStacks); err != nil {
		return nil, err
	}
	var filtered []Stack
	for _, s := range allStacks {
		if endpointID == 0 || s.EndpointID == endpointID {
			filtered = append(filtered, s)
		}
	}
	return filtered, nil
}

func (c *Client) GetStackFile(ctx context.Context, stackID int) (string, error) {
	path := fmt.Sprintf("/api/stacks/%d/file", stackID)
	resp, err := c.do(ctx, "GET", path, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var sf StackFile
	if err := json.NewDecoder(resp.Body).Decode(&sf); err != nil {
		return "", err
	}
	return sf.Content, nil
}

func (c *Client) StopStack(ctx context.Context, stackID, endpointID int) error {
	path := fmt.Sprintf("/api/stacks/%d/stop?endpointId=%d", stackID, endpointID)
	resp, err := c.do(ctx, "POST", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("stop stack: status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) StartStack(ctx context.Context, stackID, endpointID int) error {
	path := fmt.Sprintf("/api/stacks/%d/start?endpointId=%d", stackID, endpointID)
	resp, err := c.do(ctx, "POST", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("start stack: status %d", resp.StatusCode)
	}
	return nil
}

// IsDvbBacked checks if compose content references the dvb backup image.
func IsDvbBacked(composeContent string) bool {
	return strings.Contains(composeContent, "offen/docker-volume-backup")
}
