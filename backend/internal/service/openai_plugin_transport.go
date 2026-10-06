package service

import (
	"context"
	"net/http"
)

func (s *OpenAIGatewayService) SetPluginManager(manager *PluginManager) {
	s.pluginManager = manager
}

// doOpenAIUpstream 只在 OpenAI OAuth 能力绑定已启用时把真实请求交给插件。
// 插件返回标准 http.Response，响应解析、错误映射、SSE 和计费仍由现有核心链处理。
func (s *OpenAIGatewayService) doOpenAIUpstream(request *http.Request, proxyURL string, account *Account) (*http.Response, error) {
	return s.doOpenAIUpstreamWithRedirectBudget(request, proxyURL, account, ReserveSmartRoutingOpenAIRequestAttempt)
}

func (s *OpenAIGatewayService) doOpenAIWSTurnUpstream(request *http.Request, proxyURL string, account *Account) (*http.Response, error) {
	return s.doOpenAIUpstreamWithRedirectBudget(request, proxyURL, account, ReserveSmartRoutingOpenAIWSTurnAttempt)
}

func (s *OpenAIGatewayService) doOpenAIUpstreamWithRedirectBudget(
	request *http.Request,
	proxyURL string,
	account *Account,
	reserve func(context.Context, int64) error,
) (*http.Response, error) {
	if request != nil && account != nil {
		if err := reserve(request.Context(), account.ID); err != nil {
			return nil, err
		}
	}
	if s.pluginManager != nil {
		response, handled, err := s.pluginManager.RoundTripOpenAIOAuth(request.Context(), request, proxyURL, account)
		if handled {
			return response, err
		}
	}

	// The in-process HTTP client invokes this callback before each redirected
	// send. Plugins receive the original request and may use a different
	// transport, so their redirect behavior is not covered by this hook.
	if request != nil && account != nil && smartRoutingOpenAIForward(request.Context(), account) {
		accountID := account.ID
		request = request.WithContext(WithHTTPUpstreamRedirectAttemptReserver(
			request.Context(),
			func(redirectCtx context.Context) error {
				return reserve(redirectCtx, accountID)
			},
		))
	}
	return s.httpUpstream.Do(request, proxyURL, account.ID, account.Concurrency)
}

// doOpenAIAccountTestUpstream 让 OpenAI OAuth 账号测试与真实转发使用同一插件路径。
// API Key 和未命中插件的账号保持各自原有的 HTTPUpstream 行为。
func (s *AccountTestService) doOpenAIAccountTestUpstream(
	request *http.Request,
	proxyURL string,
	account *Account,
	useTLSFallback bool,
) (*http.Response, error) {
	if s.pluginManager != nil {
		response, handled, err := s.pluginManager.RoundTripOpenAIOAuth(request.Context(), request, proxyURL, account)
		if handled {
			return response, err
		}
	}
	if useTLSFallback {
		return s.httpUpstream.DoWithTLS(
			request,
			proxyURL,
			account.ID,
			account.Concurrency,
			s.tlsFPProfileService.ResolveTLSProfile(account),
		)
	}
	return s.httpUpstream.Do(request, proxyURL, account.ID, account.Concurrency)
}
