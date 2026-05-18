package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/imroc/req/v3"
)

type proxySubscriptionSidecarClient struct {
	baseURL string
	client  *req.Client
}

func NewProxySubscriptionSidecarClient(cfg *config.Config) service.ProxySubscriptionSidecarClient {
	if cfg == nil || !cfg.ProxySubscriptionSidecar.Enabled {
		return nil
	}
	return &proxySubscriptionSidecarClient{
		baseURL: strings.TrimRight(strings.TrimSpace(cfg.ProxySubscriptionSidecar.BaseURL), "/"),
		client:  req.C().SetTimeout(30 * time.Second),
	}
}

func (c *proxySubscriptionSidecarClient) HealthCheck(ctx context.Context) error {
	resp, err := c.client.R().SetContext(ctx).Get(c.baseURL + "/healthz")
	if err != nil {
		return err
	}
	if !resp.IsSuccessState() {
		return fmt.Errorf("sidecar health check failed: status %d", resp.StatusCode)
	}
	return nil
}

func (c *proxySubscriptionSidecarClient) UpsertRuntime(ctx context.Context, reqBody service.ProxySubscriptionSidecarUpsertRequest) (*service.ProxySubscriptionSidecarUpsertResponse, error) {
	var out service.ProxySubscriptionSidecarUpsertResponse
	resp, err := c.client.R().
		SetContext(ctx).
		SetBody(reqBody).
		SetSuccessResult(&out).
		Post(c.baseURL + "/v1/runtimes/upsert")
	if err != nil {
		return nil, err
	}
	if !resp.IsSuccessState() {
		return nil, fmt.Errorf("sidecar upsert failed: status %d body=%s", resp.StatusCode, resp.String())
	}
	return &out, nil
}

func (c *proxySubscriptionSidecarClient) DeleteRuntime(ctx context.Context, runtimeID string) error {
	resp, err := c.client.R().SetContext(ctx).Delete(c.baseURL + "/v1/runtimes/" + runtimeID)
	if err != nil {
		return err
	}
	if !resp.IsSuccessState() {
		return fmt.Errorf("sidecar delete failed: status %d", resp.StatusCode)
	}
	return nil
}

func (c *proxySubscriptionSidecarClient) CheckRuntime(ctx context.Context, runtimeID string) error {
	resp, err := c.client.R().SetContext(ctx).Post(c.baseURL + "/v1/runtimes/" + runtimeID + "/check")
	if err != nil {
		return err
	}
	if !resp.IsSuccessState() {
		return fmt.Errorf("sidecar runtime check failed: status %d", resp.StatusCode)
	}
	return nil
}
