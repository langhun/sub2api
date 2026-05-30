package service

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai_compat"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

// sseDataPrefix matches SSE data lines with optional whitespace after colon.
// Some upstream APIs return non-standard "data:" without space (should be "data: ").
var sseDataPrefix = regexp.MustCompile(`^data:\s*`)

const (
	testClaudeAPIURL   = "https://api.anthropic.com/v1/messages?beta=true"
	chatgptCodexAPIURL = "https://chatgpt.com/backend-api/codex/responses"
)

var openAIImageTestHeartbeatInterval = 15 * time.Second

// TestEvent represents a SSE event for account testing
type TestEvent struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Model    string `json:"model,omitempty"`
	Status   string `json:"status,omitempty"`
	Code     string `json:"code,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	Data     any    `json:"data,omitempty"`
	Success  bool   `json:"success,omitempty"`
	Error    string `json:"error,omitempty"`
}

const (
	defaultGeminiTextTestPrompt  = "hi"
	defaultGeminiImageTestPrompt = "Generate a cute orange cat astronaut sticker on a clean pastel background."
	defaultOpenAIImageTestPrompt = "Generate a cute orange cat astronaut sticker on a clean pastel background."
)

// isOpenAIImageModel checks if the model is an OpenAI image generation model (e.g. gpt-image-2).
func isOpenAIImageModel(model string) bool {
	return strings.HasPrefix(strings.ToLower(model), "gpt-image-")
}

// AccountTestService handles account testing operations
type AccountTestService struct {
	accountRepo               AccountRepository
	groupRepo                 GroupRepository
	geminiTokenProvider       *GeminiTokenProvider
	claudeTokenProvider       *ClaudeTokenProvider
	antigravityGatewayService *AntigravityGatewayService
	httpUpstream              HTTPUpstream
	cfg                       *config.Config
	tlsFPProfileService       *TLSFingerprintProfileService
	proxyPool                 *AutoFailoverProxyPoolService
}

// NewAccountTestService creates a new AccountTestService
func NewAccountTestService(
	accountRepo AccountRepository,
	groupRepo GroupRepository,
	geminiTokenProvider *GeminiTokenProvider,
	claudeTokenProvider *ClaudeTokenProvider,
	antigravityGatewayService *AntigravityGatewayService,
	httpUpstream HTTPUpstream,
	cfg *config.Config,
	tlsFPProfileService *TLSFingerprintProfileService,
	proxyPool *AutoFailoverProxyPoolService,
) *AccountTestService {
	return &AccountTestService{
		accountRepo:               accountRepo,
		groupRepo:                 groupRepo,
		geminiTokenProvider:       geminiTokenProvider,
		claudeTokenProvider:       claudeTokenProvider,
		antigravityGatewayService: antigravityGatewayService,
		httpUpstream:              httpUpstream,
		cfg:                       cfg,
		tlsFPProfileService:       tlsFPProfileService,
		proxyPool:                 proxyPool,
	}
}

func (s *AccountTestService) HasAutoFailoverProxyPool() bool {
	return s != nil && s.proxyPool != nil
}

func (s *AccountTestService) resolveTestProxyURL(ctx context.Context, account *Account) (string, error) {
	if account == nil {
		return "", nil
	}

	if s.proxyPool != nil && s.proxyPool.SupportsAccount(account) {
		proxyURL, _, _, err := s.proxyPool.ResolveProxyURL(ctx, account)
		if err != nil {
			return "", err
		}
		if proxyURL != "" {
			return proxyURL, nil
		}
		if account.UsesAutoFailoverProxyPool() {
			return "", errors.New("no available proxy in auto failover proxy pool")
		}
	}

	if account.ProxyID != nil && account.Proxy != nil {
		return account.Proxy.URL(), nil
	}

	return "", nil
}

func (s *AccountTestService) doTestRequestWithTLS(
	ctx context.Context,
	account *Account,
	req *http.Request,
	profile *tlsfingerprint.Profile,
) (*http.Response, error) {
	return s.doTestRequest(ctx, account, req, func(clonedReq *http.Request, proxyURL string) (*http.Response, error) {
		return s.httpUpstream.DoWithTLS(clonedReq, proxyURL, account.ID, account.Concurrency, profile)
	})
}

func (s *AccountTestService) doTestRequest(
	ctx context.Context,
	account *Account,
	req *http.Request,
	do func(*http.Request, string) (*http.Response, error),
) (*http.Response, error) {
	if req == nil || do == nil {
		return nil, errors.New("test request is not available")
	}

	if s.proxyPool != nil && s.proxyPool.SupportsAccount(account) {
		candidates, err := s.proxyPool.BuildCandidates(ctx, account)
		if err != nil {
			return nil, err
		}
		if account != nil && account.UsesAutoFailoverProxyPool() && len(candidates) == 0 {
			return nil, errors.New("no available proxy in auto failover proxy pool")
		}
		if len(candidates) > 0 {
			return s.proxyPool.DoHTTPRequest(ctx, account, req, do)
		}
	}

	proxyURL, err := s.resolveTestProxyURL(ctx, account)
	if err != nil {
		return nil, err
	}
	return do(req, proxyURL)
}

func (s *AccountTestService) validateUpstreamBaseURL(raw string) (string, error) {
	if s.cfg == nil {
		return "", errors.New("config is not available")
	}
	return urlvalidator.ValidateHTTPURL(raw, true, urlvalidator.ValidationOptions{})
}

func (s *AccountTestService) formatBaseURLError(err error) string {
	if err == nil {
		return ""
	}

	msg := err.Error()
	switch {
	case strings.HasPrefix(msg, "invalid url scheme: "):
		scheme := strings.TrimSpace(strings.TrimPrefix(msg, "invalid url scheme: "))
		return fmt.Sprintf("基础地址校验失败：URL 协议 `%s` 不受支持，请使用 http 或 https。", scheme)
	case msg == "invalid host":
		return "基础地址校验失败：主机名不能为空，请检查 base_url 是否填写完整。"
	case msg == "url is required":
		return "基础地址校验失败：base_url 不能为空。"
	case strings.HasPrefix(msg, "invalid url: "):
		return fmt.Sprintf("基础地址格式无效：%s。请填写完整的绝对地址，例如 `https://example.com`。", strings.TrimSpace(strings.TrimPrefix(msg, "invalid url: ")))
	case strings.HasPrefix(msg, "invalid port: "):
		return fmt.Sprintf("基础地址端口无效：%s。请检查 URL 中的端口号。", strings.TrimSpace(strings.TrimPrefix(msg, "invalid port: ")))
	default:
		return fmt.Sprintf("基础地址校验失败：%s", msg)
	}
}

func (s *AccountTestService) formatTestErrorMessage(errorMsg string) string {
	msg := strings.TrimSpace(errorMsg)
	if msg == "" {
		return msg
	}

	switch {
	case strings.HasPrefix(msg, "基础地址"):
		return msg
	case msg == "Account not found":
		return "未找到要测试的账号。"
	case msg == "No access token available":
		return "账号缺少可用的访问令牌，请先完成授权或检查 access_token。"
	case msg == "No API key available":
		return "账号缺少可用的 API Key，请先填写并保存 API Key。"
	case strings.HasPrefix(msg, "Unsupported account type: "):
		return "当前账号类型不支持此测试方式：" + strings.TrimSpace(strings.TrimPrefix(msg, "Unsupported account type: "))
	case strings.HasPrefix(msg, "Unsupported Bedrock model: "):
		return "当前 Bedrock 模型暂不支持测试：" + strings.TrimSpace(strings.TrimPrefix(msg, "Unsupported Bedrock model: "))
	case msg == "Failed to create test payload":
		return "创建测试请求体失败。"
	case strings.HasPrefix(msg, "Failed to create request"):
		return "创建测试请求失败。"
	case strings.HasPrefix(msg, "Failed to create Chat Completions request"):
		return "创建 Chat Completions 测试请求失败。"
	case strings.HasPrefix(msg, "Failed to build request: "):
		return "构建测试请求失败：" + strings.TrimSpace(strings.TrimPrefix(msg, "Failed to build request: "))
	case strings.HasPrefix(msg, "Failed to build image request: "):
		return "构建图片测试请求失败：" + strings.TrimSpace(strings.TrimPrefix(msg, "Failed to build image request: "))
	case strings.HasPrefix(msg, "Failed to create Vertex request body: "):
		return "构建 Vertex 请求体失败：" + strings.TrimSpace(strings.TrimPrefix(msg, "Failed to create Vertex request body: "))
	case strings.HasPrefix(msg, "Failed to build Vertex URL: "):
		return "构建 Vertex 请求地址失败：" + strings.TrimSpace(strings.TrimPrefix(msg, "Failed to build Vertex URL: "))
	case msg == "Claude token provider not configured":
		return "Claude Token 提供器未配置，无法测试该账号。"
	case msg == "Antigravity gateway service not configured":
		return "Antigravity 网关服务未配置，无法测试该账号。"
	case strings.HasPrefix(msg, "Failed to get service account access token: "):
		return "获取服务账号访问令牌失败：" + strings.TrimSpace(strings.TrimPrefix(msg, "Failed to get service account access token: "))
	case strings.HasPrefix(msg, "Failed to get access token: "):
		return "获取访问令牌失败：" + strings.TrimSpace(strings.TrimPrefix(msg, "Failed to get access token: "))
	case strings.HasPrefix(msg, "Failed to create Bedrock signer: "):
		return "创建 Bedrock 签名器失败：" + strings.TrimSpace(strings.TrimPrefix(msg, "Failed to create Bedrock signer: "))
	case strings.HasPrefix(msg, "Failed to sign request: "):
		return "签名请求失败：" + strings.TrimSpace(strings.TrimPrefix(msg, "Failed to sign request: "))
	case strings.HasPrefix(msg, "Failed to resolve proxy: "):
		return "解析代理失败：" + strings.TrimSpace(strings.TrimPrefix(msg, "Failed to resolve proxy: "))
	case strings.HasPrefix(msg, "Request failed: "):
		return "请求失败：" + strings.TrimSpace(strings.TrimPrefix(msg, "Request failed: "))
	case strings.HasPrefix(msg, "Responses API request failed: "):
		return "Responses API 请求失败：" + strings.TrimSpace(strings.TrimPrefix(msg, "Responses API request failed: "))
	case strings.HasPrefix(msg, "Chat Completions API (/v1/chat/completions) request failed: "):
		return "Chat Completions API（/v1/chat/completions）请求失败：" + strings.TrimSpace(strings.TrimPrefix(msg, "Chat Completions API (/v1/chat/completions) request failed: "))
	case strings.HasPrefix(msg, "Failed to read response: "):
		return "读取上游响应失败：" + strings.TrimSpace(strings.TrimPrefix(msg, "Failed to read response: "))
	case strings.HasPrefix(msg, "Failed to parse response: "):
		return "解析上游响应失败：" + strings.TrimSpace(strings.TrimPrefix(msg, "Failed to parse response: "))
	case strings.HasPrefix(msg, "Stream read error: "):
		return "读取流式响应失败：" + strings.TrimSpace(strings.TrimPrefix(msg, "Stream read error: "))
	case strings.HasPrefix(msg, "Stream write error: "):
		return "写入流式响应失败：" + strings.TrimSpace(strings.TrimPrefix(msg, "Stream write error: "))
	case msg == "Stream ended before response.completed":
		return "流式响应在收到 `response.completed` 之前就结束了。"
	case msg == "Stream ended before completion":
		return "流式响应在完成前就中断了。"
	case strings.HasPrefix(msg, "Authentication failed (401): "):
		return "鉴权失败（401）：" + strings.TrimSpace(strings.TrimPrefix(msg, "Authentication failed (401): "))
	case strings.HasPrefix(msg, "Chat Completions authentication failed (401): "):
		return "Chat Completions 鉴权失败（401）：" + strings.TrimSpace(strings.TrimPrefix(msg, "Chat Completions authentication failed (401): "))
	case strings.HasPrefix(msg, "API returned "):
		return "上游接口返回错误：" + strings.TrimSpace(strings.TrimPrefix(msg, "API returned "))
	case strings.HasPrefix(msg, "Chat Completions API (/v1/chat/completions) returned "):
		return "Chat Completions API 返回错误：" + strings.TrimSpace(strings.TrimPrefix(msg, "Chat Completions API (/v1/chat/completions) returned "))
	case msg == "No images returned from API":
		return "上游接口没有返回图片结果。"
	case strings.HasPrefix(msg, "Failed to bind default group: "):
		return "绑定默认分组失败：" + strings.TrimSpace(strings.TrimPrefix(msg, "Failed to bind default group: "))
	case msg == "OpenAI response failed":
		return "OpenAI 响应失败。"
	case msg == "Unknown error":
		return "发生未知错误。"
	default:
		return msg
	}
}

func (s *AccountTestService) tryEnableOpenAIProxyPoolForTest(ctx context.Context, account *Account) (bool, error) {
	if s == nil || s.accountRepo == nil || s.proxyPool == nil || account == nil || account.Platform != PlatformOpenAI {
		return false, nil
	}
	if normalizeAccountProxyMode(account.GetExtraString("proxy_mode")) == AccountProxyModePool {
		return false, nil
	}

	probe := *account
	probe.Extra = copyAnyMap(account.Extra)
	if probe.Extra == nil {
		probe.Extra = map[string]any{}
	}
	probe.Extra["proxy_mode"] = AccountProxyModePool

	candidates, err := s.proxyPool.BuildCandidates(ctx, &probe)
	if err != nil {
		return false, fmt.Errorf("build proxy pool candidates: %w", err)
	}
	if len(candidates) == 0 {
		return false, nil
	}

	if account.Extra == nil {
		account.Extra = map[string]any{}
	}
	account.Extra["proxy_mode"] = AccountProxyModePool
	if err := s.accountRepo.UpdateExtra(ctx, account.ID, map[string]any{"proxy_mode": AccountProxyModePool}); err != nil {
		return false, fmt.Errorf("persist proxy_mode=pool: %w", err)
	}
	return true, nil
}

// generateSessionString generates a Claude Code style session string.
// The output format is determined by the UA version in claude.DefaultHeaders,
// ensuring consistency between the user_id format and the UA sent to upstream.
func generateSessionString() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	hex64 := hex.EncodeToString(b)
	sessionUUID := uuid.New().String()
	uaVersion := ExtractCLIVersion(claude.DefaultHeaders["User-Agent"])
	return FormatMetadataUserID(hex64, "", sessionUUID, uaVersion), nil
}

// createTestPayload creates a Claude Code style test request payload
func createTestPayload(modelID string) (map[string]any, error) {
	sessionID, err := generateSessionString()
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"model": modelID,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "text",
						"text": "hi",
						"cache_control": map[string]string{
							"type": "ephemeral",
						},
					},
				},
			},
		},
		"system": []map[string]any{
			{
				"type": "text",
				"text": claudeCodeSystemPrompt,
				"cache_control": map[string]string{
					"type": "ephemeral",
				},
			},
		},
		"metadata": map[string]string{
			"user_id": sessionID,
		},
		"max_tokens":  1024,
		"temperature": 1,
		"stream":      true,
	}, nil
}

// TestAccountConnection tests an account's connection by sending a test request.
// All account types use full Claude Code client characteristics, only auth header differs.
// modelID is optional - if empty, defaults to the platform test model.
func (s *AccountTestService) TestAccountConnection(c *gin.Context, accountID int64, modelID string, prompt string, mode string) error {
	ctx := c.Request.Context()

	// Get account
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return s.sendErrorAndEnd(c, "Account not found")
	}

	// Route to platform-specific test method
	if account.IsOpenAI() {
		return s.testOpenAIAccountConnection(c, account, modelID, prompt)
	}

	if account.IsGemini() {
		return s.testGeminiAccountConnection(c, account, modelID, prompt)
	}

	if account.Platform == PlatformAntigravity {
		return s.routeAntigravityTest(c, account, modelID, prompt)
	}

	return s.testClaudeAccountConnection(c, account, modelID)
}

// testClaudeAccountConnection tests an Anthropic Claude account's connection
func (s *AccountTestService) testClaudeAccountConnection(c *gin.Context, account *Account, modelID string) error {
	ctx := c.Request.Context()

	// Determine the model to use
	testModelID := modelID
	if testModelID == "" {
		testModelID = claude.DefaultTestModel
	}

	// API Key 账号测试连接时也需要应用通配符模型映射。
	if account.Type == "apikey" {
		testModelID = account.GetMappedModel(testModelID)
	}

	// Bedrock accounts use a separate test path
	if account.IsBedrock() {
		return s.testBedrockAccountConnection(c, ctx, account, testModelID)
	}
	if account.Type == AccountTypeServiceAccount {
		return s.testClaudeVertexServiceAccountConnection(c, ctx, account, testModelID)
	}

	// Determine authentication method and API URL
	var authToken string
	var useBearer bool
	var apiURL string

	if account.IsOAuth() {
		// OAuth or Setup Token - use Bearer token
		useBearer = true
		apiURL = testClaudeAPIURL
		authToken = account.GetCredential("access_token")
		if authToken == "" {
			return s.sendErrorAndEnd(c, "No access token available")
		}
	} else if account.Type == "apikey" {
		// API Key - use x-api-key header
		useBearer = false
		authToken = account.GetCredential("api_key")
		if authToken == "" {
			return s.sendErrorAndEnd(c, "No API key available")
		}

		baseURL := account.GetBaseURL()
		if baseURL == "" {
			baseURL = "https://api.anthropic.com"
		}
		normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return s.sendErrorAndEnd(c, s.formatBaseURLError(err))
		}
		apiURL = strings.TrimSuffix(normalizedBaseURL, "/") + "/v1/messages?beta=true"
	} else {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Unsupported account type: %s", account.Type))
	}

	// Set SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	// Create Claude Code style payload (same for all account types)
	payload, err := createTestPayload(testModelID)
	if err != nil {
		return s.sendErrorAndEnd(c, "Failed to create test payload")
	}
	payloadBytes, _ := json.Marshal(payload)

	// Send test_start event
	s.sendEvent(c, TestEvent{Type: "test_start", Model: testModelID})

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return s.sendErrorAndEnd(c, "Failed to create request")
	}

	// Set common headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")

	// Apply Claude Code client headers
	for key, value := range claude.DefaultHeaders {
		req.Header.Set(key, value)
	}

	// Set authentication header
	if useBearer {
		req.Header.Set("anthropic-beta", claude.DefaultBetaHeader)
		req.Header.Set("Authorization", "Bearer "+authToken)
	} else {
		req.Header.Set("anthropic-beta", claude.APIKeyBetaHeader)
		req.Header.Set("x-api-key", authToken)
	}

	resp, err := s.doTestRequestWithTLS(ctx, account, req, s.tlsFPProfileService.ResolveTLSProfile(account))
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Request failed: %s", err.Error()))
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		errMsg := fmt.Sprintf("API returned %d: %s", resp.StatusCode, string(body))

		// 403 表示账号被上游封禁，标记为 error 状态
		if resp.StatusCode == http.StatusForbidden {
			_ = s.accountRepo.SetError(ctx, account.ID, errMsg)
		}

		return s.sendErrorAndEnd(c, errMsg)
	}

	// Process SSE stream
	return s.processClaudeStream(c, account, resp.Body)
}

func (s *AccountTestService) testClaudeVertexServiceAccountConnection(c *gin.Context, ctx context.Context, account *Account, testModelID string) error {
	if mappedModel, matched := account.ResolveMappedModel(testModelID); matched {
		testModelID = mappedModel
	} else {
		testModelID = normalizeVertexAnthropicModelID(claude.NormalizeModelID(testModelID))
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	payload, err := createTestPayload(testModelID)
	if err != nil {
		return s.sendErrorAndEnd(c, "Failed to create test payload")
	}
	payloadBytes, _ := json.Marshal(payload)
	vertexBody, err := buildVertexAnthropicRequestBody(payloadBytes)
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Failed to create Vertex request body: %s", err.Error()))
	}

	if s.claudeTokenProvider == nil {
		return s.sendErrorAndEnd(c, "Claude token provider not configured")
	}
	accessToken, err := s.claudeTokenProvider.GetAccessToken(ctx, account)
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Failed to get service account access token: %s", err.Error()))
	}

	fullURL, err := buildVertexAnthropicURL(account.VertexProjectID(), account.VertexLocation(testModelID), testModelID, true)
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Failed to build Vertex URL: %s", err.Error()))
	}

	s.sendEvent(c, TestEvent{Type: "test_start", Model: testModelID})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(vertexBody))
	if err != nil {
		return s.sendErrorAndEnd(c, "Failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := s.doTestRequestWithTLS(ctx, account, req, s.tlsFPProfileService.ResolveTLSProfile(account))
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Request failed: %s", err.Error()))
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		errMsg := fmt.Sprintf("API returned %d: %s", resp.StatusCode, string(body))
		if resp.StatusCode == http.StatusForbidden {
			_ = s.accountRepo.SetError(ctx, account.ID, errMsg)
		}
		return s.sendErrorAndEnd(c, errMsg)
	}

	return s.processClaudeStream(c, account, resp.Body)
}

// testBedrockAccountConnection tests a Bedrock (SigV4 or API Key) account using non-streaming invoke
func (s *AccountTestService) testBedrockAccountConnection(c *gin.Context, ctx context.Context, account *Account, testModelID string) error {
	region := bedrockRuntimeRegion(account)
	resolvedModelID, ok := ResolveBedrockModelID(account, testModelID)
	if !ok {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Unsupported Bedrock model: %s", testModelID))
	}
	testModelID = resolvedModelID

	// Set SSE headers (test UI expects SSE)
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	// Create a minimal Bedrock-compatible payload (no stream, no cache_control)
	bedrockPayload := map[string]any{
		"anthropic_version": "bedrock-2023-05-31",
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "text",
						"text": "hi",
					},
				},
			},
		},
		"max_tokens":  256,
		"temperature": 1,
	}
	bedrockBody, _ := json.Marshal(bedrockPayload)

	// Use non-streaming endpoint (response is standard Claude JSON)
	apiURL := BuildBedrockURL(region, testModelID, false)

	s.sendEvent(c, TestEvent{Type: "test_start", Model: testModelID})

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(bedrockBody))
	if err != nil {
		return s.sendErrorAndEnd(c, "Failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")

	// Sign or set auth based on account type
	if account.IsBedrockAPIKey() {
		apiKey := account.GetCredential("api_key")
		if apiKey == "" {
			return s.sendErrorAndEnd(c, "No API key available")
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
	} else {
		signer, err := NewBedrockSignerFromAccount(account)
		if err != nil {
			return s.sendErrorAndEnd(c, fmt.Sprintf("Failed to create Bedrock signer: %s", err.Error()))
		}
		if err := signer.SignRequest(ctx, req, bedrockBody); err != nil {
			return s.sendErrorAndEnd(c, fmt.Sprintf("Failed to sign request: %s", err.Error()))
		}
	}

	resp, err := s.doTestRequestWithTLS(ctx, account, req, nil)
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Request failed: %s", err.Error()))
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return s.sendHTTPErrorAndEnd(c, resp.StatusCode, fmt.Sprintf("API returned %d: %s", resp.StatusCode, string(body)))
	}

	// Bedrock non-streaming response is standard Claude JSON, extract the text
	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Failed to parse response: %s", err.Error()))
	}

	text := ""
	if len(result.Content) > 0 {
		text = result.Content[0].Text
	}
	if text == "" {
		text = "(empty response)"
	}

	s.sendEvent(c, TestEvent{Type: "content", Text: text})
	return s.completeSuccessfulTest(c, account)
}

// testOpenAIAccountConnection tests an OpenAI account's connection
func (s *AccountTestService) testOpenAIAccountConnection(c *gin.Context, account *Account, modelID string, prompt string) error {
	ctx := c.Request.Context()
	_ = prompt

	// Default to openai.DefaultTestModel for OpenAI testing
	testModelID := modelID
	if testModelID == "" {
		testModelID = openai.DefaultTestModel
	}

	// Align test routing with gateway behavior: OpenAI accounts apply normal
	// account model mapping.
	testModelID = account.GetMappedModel(testModelID)

	// Route to image generation test if an image model is selected
	if isOpenAIImageModel(testModelID) {
		imagePrompt := strings.TrimSpace(prompt)
		if imagePrompt == "" {
			imagePrompt = defaultOpenAIImageTestPrompt
		}
		if account.Type == "apikey" {
			return s.testOpenAIImageAPIKey(c, ctx, account, testModelID, imagePrompt)
		}
		return s.testOpenAIImageOAuth(c, ctx, account, testModelID, imagePrompt)
	}

	// Determine authentication method and API URL
	var authToken string
	var apiURL string
	var isOAuth bool
	var chatgptAccountID string
	var useCCChatCompletions bool

	if account.IsOAuth() {
		isOAuth = true
		// OAuth - use Bearer token with ChatGPT internal API
		authToken = account.GetOpenAIAccessToken()
		if authToken == "" {
			return s.sendErrorAndEnd(c, "No access token available")
		}

		// OAuth uses ChatGPT internal API
		apiURL = chatgptCodexAPIURL
		chatgptAccountID = account.GetChatGPTAccountID()
	} else if account.Type == "apikey" {
		// API Key - use Platform API
		authToken = account.GetOpenAIApiKey()
		if authToken == "" {
			return s.sendErrorAndEnd(c, "No API key available")
		}

		baseURL := account.GetOpenAIBaseURL()
		if baseURL == "" {
			baseURL = "https://api.openai.com"
		}
		normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return s.sendErrorAndEnd(c, s.formatBaseURLError(err))
		}
		if !openai_compat.ShouldUseResponsesAPI(account.Extra) {
			apiURL = buildOpenAIChatCompletionsURL(normalizedBaseURL)
			useCCChatCompletions = true
		} else {
			apiURL = buildOpenAIResponsesURL(normalizedBaseURL)
		}
	} else {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Unsupported account type: %s", account.Type))
	}

	// Set SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	payload := createOpenAITestPayload(testModelID, isOAuth)
	if useCCChatCompletions {
		payload = createOpenAIChatCompletionsTestPayload(testModelID)
	}
	payloadBytes, _ := json.Marshal(payload)

	// Send test_start event
	s.sendEvent(c, TestEvent{Type: "test_start", Model: testModelID})

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return s.sendErrorAndEnd(c, "Failed to create request")
	}
	req = req.WithContext(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI))

	// Set common headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)
	if useCCChatCompletions {
		req.Header.Set("Accept", "text/event-stream")
	}

	// Set OAuth-specific headers for ChatGPT internal API
	if isOAuth {
		req.Host = "chatgpt.com"
		req.Header.Set("accept", "text/event-stream")
		if chatgptAccountID != "" {
			req.Header.Set("chatgpt-account-id", chatgptAccountID)
		}
	}

	autoPoolSwitched := false
	for {
		resp, err := s.doTestRequestWithTLS(ctx, account, req, s.tlsFPProfileService.ResolveTLSProfile(account))
		if err != nil {
			return s.sendErrorAndEnd(c, fmt.Sprintf("Request failed: %s", err.Error()))
		}

		if isOAuth && s.accountRepo != nil {
			if updates, err := extractOpenAICodexProbeUpdates(resp); err == nil && len(updates) > 0 {
				_ = s.accountRepo.UpdateExtra(ctx, account.ID, updates)
				mergeAccountExtra(account, updates)
			}
		}

		if resp.StatusCode == http.StatusOK {
			if useCCChatCompletions {
				err = s.processOpenAIChatCompletionsStream(c, account, resp.Body)
			} else {
				err = s.processOpenAIStream(c, account, resp.Body)
			}
			_ = resp.Body.Close()
			return err
		}

		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode == http.StatusForbidden && !autoPoolSwitched {
			switched, switchErr := s.tryEnableOpenAIProxyPoolForTest(ctx, account)
			if switchErr != nil {
				return s.sendHTTPErrorAndEnd(c, resp.StatusCode, fmt.Sprintf("API returned %d: %s; auto switch to proxy pool failed: %s", resp.StatusCode, string(body), switchErr.Error()))
			}
			if switched {
				autoPoolSwitched = true
				continue
			}
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			s.reconcileOpenAI429State(ctx, account, resp.Header, body)
		}
		// 401 Unauthorized: 标记账号为永久错误
		if resp.StatusCode == http.StatusUnauthorized && s.accountRepo != nil {
			errMsg := fmt.Sprintf("Authentication failed (401): %s", string(body))
			_ = s.accountRepo.SetError(ctx, account.ID, errMsg)
		}
		return s.sendHTTPErrorAndEnd(c, resp.StatusCode, fmt.Sprintf("API returned %d: %s", resp.StatusCode, string(body)))
	}
}

// testOpenAIChatCompletionsConnection tests an OpenAI-compatible APIKey account
// through the raw /v1/chat/completions endpoint.
func (s *AccountTestService) testOpenAIChatCompletionsConnection(
	c *gin.Context,
	account *Account,
	testModelID string,
	prompt string,
	normalizedBaseURL string,
	authToken string,
) error {
	ctx := c.Request.Context()
	apiURL := buildOpenAIChatCompletionsURL(normalizedBaseURL)

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	payload := createOpenAIChatCompletionsTestPayload(testModelID)
	payloadBytes, _ := json.Marshal(payload)

	s.sendEvent(c, TestEvent{Type: "test_start", Model: testModelID})
	s.sendEvent(c, TestEvent{Type: "status", Text: "正在通过 /v1/chat/completions 测试连接"})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return s.sendErrorAndEnd(c, "Failed to create Chat Completions request")
	}
	req = req.WithContext(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Authorization", "Bearer "+authToken)

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	resp, err := s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, s.tlsFPProfileService.ResolveTLSProfile(account))
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Chat Completions API (/v1/chat/completions) request failed: %s", err.Error()))
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusTooManyRequests {
			s.reconcileOpenAI429State(ctx, account, resp.Header, body)
		}
		if resp.StatusCode == http.StatusUnauthorized && s.accountRepo != nil {
			errMsg := fmt.Sprintf("Chat Completions authentication failed (401): %s", string(body))
			_ = s.accountRepo.SetError(ctx, account.ID, errMsg)
		}
		return s.sendErrorAndEnd(c, fmt.Sprintf("Chat Completions API (/v1/chat/completions) returned %d: %s", resp.StatusCode, string(body)))
	}

	return s.processOpenAIChatCompletionsStream(c, account, resp.Body)
}

func (s *AccountTestService) reconcileOpenAI429State(ctx context.Context, account *Account, headers http.Header, body []byte) {
	if s == nil || s.accountRepo == nil || account == nil {
		return
	}

	persistOpenAI429PlanType(ctx, s.accountRepo, account, body)

	var resetAt *time.Time
	if calculated := calculateOpenAI429ResetTime(headers); calculated != nil {
		resetAt = calculated
	} else if unixTs := parseOpenAIRateLimitResetTime(body); unixTs != nil {
		t := time.Unix(*unixTs, 0)
		resetAt = &t
	} else {
		fallback := time.Now().Add(time.Duration(clampRateLimit429CooldownSeconds(defaultRateLimit429CooldownSeconds)) * time.Second)
		resetAt = &fallback
	}

	if err := s.accountRepo.SetRateLimited(ctx, account.ID, *resetAt); err != nil {
		return
	}

	now := time.Now()
	account.RateLimitedAt = &now
	account.RateLimitResetAt = resetAt

	if account.Status == StatusError {
		if err := s.accountRepo.ClearError(ctx, account.ID); err != nil {
			return
		}
		account.Status = StatusActive
		account.ErrorMessage = ""
	}
}

// testGeminiAccountConnection tests a Gemini account's connection
func (s *AccountTestService) testGeminiAccountConnection(c *gin.Context, account *Account, modelID string, prompt string) error {
	ctx := c.Request.Context()

	// Determine the model to use
	testModelID := modelID
	if testModelID == "" {
		testModelID = geminicli.DefaultTestModel
	}

	// For static upstream credentials with model mapping, map the model
	if account.Type == AccountTypeAPIKey || account.Type == AccountTypeServiceAccount {
		mapping := account.GetModelMapping()
		if len(mapping) > 0 {
			if mappedModel, exists := mapping[testModelID]; exists {
				testModelID = mappedModel
			}
		}
	}

	// Set SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	// Create test payload (Gemini format)
	payload := createGeminiTestPayload(testModelID, prompt)

	// Build request based on account type
	var req *http.Request
	var err error

	switch account.Type {
	case AccountTypeAPIKey:
		req, err = s.buildGeminiAPIKeyRequest(ctx, account, testModelID, payload)
	case AccountTypeOAuth:
		req, err = s.buildGeminiOAuthRequest(ctx, account, testModelID, payload)
	case AccountTypeServiceAccount:
		req, err = s.buildGeminiServiceAccountRequest(ctx, account, testModelID, payload)
	default:
		return s.sendErrorAndEnd(c, fmt.Sprintf("Unsupported account type: %s", account.Type))
	}

	if err != nil {
		return s.sendErrorAndEnd(c, s.formatBaseURLError(err))
	}

	// Send test_start event
	s.sendEvent(c, TestEvent{Type: "test_start", Model: testModelID})

	// Get proxy and execute request
	resp, err := s.doTestRequestWithTLS(ctx, account, req, s.tlsFPProfileService.ResolveTLSProfile(account))
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Request failed: %s", err.Error()))
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return s.sendErrorAndEnd(c, fmt.Sprintf("API returned %d: %s", resp.StatusCode, string(body)))
	}

	// Process SSE stream
	return s.processGeminiStream(c, account, resp.Body)
}

// routeAntigravityTest 路由 Antigravity 账号的测试请求。
// APIKey 类型走原生协议（与 gateway_handler 路由一致），OAuth/Upstream 走 CRS 中转。
func (s *AccountTestService) routeAntigravityTest(c *gin.Context, account *Account, modelID string, prompt string) error {
	if account.Type == AccountTypeAPIKey {
		if strings.HasPrefix(modelID, "gemini-") {
			return s.testGeminiAccountConnection(c, account, modelID, prompt)
		}
		return s.testClaudeAccountConnection(c, account, modelID)
	}
	return s.testAntigravityAccountConnection(c, account, modelID)
}

// testAntigravityAccountConnection tests an Antigravity account's connection
// 支持 Claude 和 Gemini 两种协议，使用非流式请求
func (s *AccountTestService) testAntigravityAccountConnection(c *gin.Context, account *Account, modelID string) error {
	ctx := c.Request.Context()

	// 默认模型：Claude 使用 claude-sonnet-4-5，Gemini 使用 gemini-3-pro-preview
	testModelID := modelID
	if testModelID == "" {
		testModelID = "claude-sonnet-4-5"
	}

	if s.antigravityGatewayService == nil {
		return s.sendErrorAndEnd(c, "Antigravity gateway service not configured")
	}

	// Set SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	// Send test_start event
	s.sendEvent(c, TestEvent{Type: "test_start", Model: testModelID})

	// 调用 AntigravityGatewayService.TestConnection（复用协议转换逻辑）
	result, err := s.antigravityGatewayService.TestConnection(ctx, account, testModelID)
	if err != nil {
		return s.sendErrorAndEnd(c, err.Error())
	}

	// 发送响应内容
	if result.Text != "" {
		s.sendEvent(c, TestEvent{Type: "content", Text: result.Text})
	}

	return s.completeSuccessfulTest(c, account)
}

// buildGeminiAPIKeyRequest builds request for Gemini API Key accounts
func (s *AccountTestService) buildGeminiAPIKeyRequest(ctx context.Context, account *Account, modelID string, payload []byte) (*http.Request, error) {
	apiKey := account.GetCredential("api_key")
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("no API key available")
	}

	baseURL := account.GetCredential("base_url")
	if baseURL == "" {
		baseURL = geminicli.AIStudioBaseURL
	}
	normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
	if err != nil {
		return nil, err
	}

	// Use streamGenerateContent for real-time feedback
	fullURL := fmt.Sprintf("%s/v1beta/models/%s:streamGenerateContent?alt=sse",
		strings.TrimRight(normalizedBaseURL, "/"), modelID)

	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", apiKey)

	return req, nil
}

// buildGeminiOAuthRequest builds request for Gemini OAuth accounts
func (s *AccountTestService) buildGeminiOAuthRequest(ctx context.Context, account *Account, modelID string, payload []byte) (*http.Request, error) {
	if s.geminiTokenProvider == nil {
		return nil, fmt.Errorf("gemini token provider not configured")
	}

	// Get access token (auto-refreshes if needed)
	accessToken, err := s.geminiTokenProvider.GetAccessToken(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	projectID := strings.TrimSpace(account.GetCredential("project_id"))
	if projectID == "" {
		// AI Studio OAuth mode (no project_id): call generativelanguage API directly with Bearer token.
		baseURL := account.GetCredential("base_url")
		if strings.TrimSpace(baseURL) == "" {
			baseURL = geminicli.AIStudioBaseURL
		}
		normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return nil, err
		}
		fullURL := fmt.Sprintf("%s/v1beta/models/%s:streamGenerateContent?alt=sse", strings.TrimRight(normalizedBaseURL, "/"), modelID)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+accessToken)
		return req, nil
	}

	// Code Assist mode (with project_id)
	return s.buildCodeAssistRequest(ctx, accessToken, projectID, modelID, payload)
}

func (s *AccountTestService) buildGeminiServiceAccountRequest(ctx context.Context, account *Account, modelID string, payload []byte) (*http.Request, error) {
	if s.geminiTokenProvider == nil {
		return nil, fmt.Errorf("gemini token provider not configured")
	}
	accessToken, err := s.geminiTokenProvider.GetAccessToken(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("failed to get service account access token: %w", err)
	}
	fullURL, err := buildVertexGeminiURL(account.VertexProjectID(), account.VertexLocation(modelID), modelID, "streamGenerateContent", true)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	return req, nil
}

// buildCodeAssistRequest builds request for Google Code Assist API (used by Gemini CLI and Antigravity)
func (s *AccountTestService) buildCodeAssistRequest(ctx context.Context, accessToken, projectID, modelID string, payload []byte) (*http.Request, error) {
	var inner map[string]any
	if err := json.Unmarshal(payload, &inner); err != nil {
		return nil, err
	}

	wrapped := map[string]any{
		"model":   modelID,
		"project": projectID,
		"request": inner,
	}
	wrappedBytes, _ := json.Marshal(wrapped)

	normalizedBaseURL, err := s.validateUpstreamBaseURL(geminicli.GeminiCliBaseURL)
	if err != nil {
		return nil, err
	}
	fullURL := fmt.Sprintf("%s/v1internal:streamGenerateContent?alt=sse", normalizedBaseURL)

	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, bytes.NewReader(wrappedBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("User-Agent", geminicli.GeminiCLIUserAgent)

	return req, nil
}

// createGeminiTestPayload creates a minimal test payload for Gemini API.
// Image models use the image-generation path so the frontend can preview the returned image.
func createGeminiTestPayload(modelID string, prompt string) []byte {
	if isImageGenerationModel(modelID) {
		imagePrompt := strings.TrimSpace(prompt)
		if imagePrompt == "" {
			imagePrompt = defaultGeminiImageTestPrompt
		}

		payload := map[string]any{
			"contents": []map[string]any{
				{
					"role": "user",
					"parts": []map[string]any{
						{"text": imagePrompt},
					},
				},
			},
			"generationConfig": map[string]any{
				"responseModalities": []string{"TEXT", "IMAGE"},
				"imageConfig": map[string]any{
					"aspectRatio": "1:1",
				},
			},
		}
		bytes, _ := json.Marshal(payload)
		return bytes
	}

	textPrompt := strings.TrimSpace(prompt)
	if textPrompt == "" {
		textPrompt = defaultGeminiTextTestPrompt
	}

	payload := map[string]any{
		"contents": []map[string]any{
			{
				"role": "user",
				"parts": []map[string]any{
					{"text": textPrompt},
				},
			},
		},
		"systemInstruction": map[string]any{
			"parts": []map[string]any{
				{"text": "You are a helpful AI assistant."},
			},
		},
	}
	bytes, _ := json.Marshal(payload)
	return bytes
}

// processGeminiStream processes SSE stream from Gemini API
func (s *AccountTestService) processGeminiStream(c *gin.Context, account *Account, body io.Reader) error {
	reader := bufio.NewReader(body)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return s.completeSuccessfulTest(c, account)
			}
			return s.sendErrorAndEnd(c, fmt.Sprintf("Stream read error: %s", err.Error()))
		}

		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		jsonStr := strings.TrimPrefix(line, "data: ")
		if jsonStr == "[DONE]" {
			return s.completeSuccessfulTest(c, account)
		}

		var data map[string]any
		if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
			continue
		}

		// Support two Gemini response formats:
		// - AI Studio: {"candidates": [...]}
		// - Gemini CLI: {"response": {"candidates": [...]}}
		if resp, ok := data["response"].(map[string]any); ok && resp != nil {
			data = resp
		}
		if candidates, ok := data["candidates"].([]any); ok && len(candidates) > 0 {
			if candidate, ok := candidates[0].(map[string]any); ok {
				// Extract content first (before checking completion)
				if content, ok := candidate["content"].(map[string]any); ok {
					if parts, ok := content["parts"].([]any); ok {
						for _, part := range parts {
							if partMap, ok := part.(map[string]any); ok {
								if text, ok := partMap["text"].(string); ok && text != "" {
									s.sendEvent(c, TestEvent{Type: "content", Text: text})
								}
								if inlineData, ok := partMap["inlineData"].(map[string]any); ok {
									mimeType, _ := inlineData["mimeType"].(string)
									data, _ := inlineData["data"].(string)
									if strings.HasPrefix(strings.ToLower(mimeType), "image/") && data != "" {
										s.sendEvent(c, TestEvent{
											Type:     "image",
											ImageURL: fmt.Sprintf("data:%s;base64,%s", mimeType, data),
											MimeType: mimeType,
										})
									}
								}
							}
						}
					}
				}

				// Check for completion after extracting content
				if finishReason, ok := candidate["finishReason"].(string); ok && finishReason != "" {
					return s.completeSuccessfulTest(c, account)
				}
			}
		}

		// Handle errors
		if errData, ok := data["error"].(map[string]any); ok {
			errorMsg := "Unknown error"
			if msg, ok := errData["message"].(string); ok {
				errorMsg = msg
			}
			return s.sendErrorAndEnd(c, errorMsg)
		}
	}
}

// createOpenAITestPayload creates a test payload for OpenAI Responses API
func createOpenAITestPayload(modelID string, isOAuth bool) map[string]any {
	payload := map[string]any{
		"model": modelID,
		"input": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "input_text",
						"text": "hi",
					},
				},
			},
		},
		"stream": true,
	}

	// OAuth accounts using ChatGPT internal API require store: false
	if isOAuth {
		payload["store"] = false
	}

	// All accounts require instructions for Responses API
	payload["instructions"] = openai.DefaultInstructions

	return payload
}

// createOpenAIChatCompletionsTestPayload creates a minimal test payload for
// OpenAI-compatible chat/completions upstreams (CC passthrough path).
func createOpenAIChatCompletionsTestPayload(modelID string) map[string]any {
	return map[string]any{
		"model": modelID,
		"messages": []map[string]any{
			{
				"role":    "user",
				"content": "hi",
			},
		},
		"stream": true,
	}
}

// processClaudeStream processes the SSE stream from Claude API
func (s *AccountTestService) processClaudeStream(c *gin.Context, account *Account, body io.Reader) error {
	reader := bufio.NewReader(body)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return s.completeSuccessfulTest(c, account)
			}
			return s.sendErrorAndEnd(c, fmt.Sprintf("Stream read error: %s", err.Error()))
		}

		line = strings.TrimSpace(line)
		if line == "" || !sseDataPrefix.MatchString(line) {
			continue
		}

		jsonStr := sseDataPrefix.ReplaceAllString(line, "")
		if jsonStr == "[DONE]" {
			return s.completeSuccessfulTest(c, account)
		}

		var data map[string]any
		if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
			continue
		}

		eventType, _ := data["type"].(string)

		switch eventType {
		case "content_block_delta":
			if delta, ok := data["delta"].(map[string]any); ok {
				if text, ok := delta["text"].(string); ok {
					s.sendEvent(c, TestEvent{Type: "content", Text: text})
				}
			}
		case "message_stop":
			return s.completeSuccessfulTest(c, account)
		case "error":
			errorMsg := "Unknown error"
			if errData, ok := data["error"].(map[string]any); ok {
				if msg, ok := errData["message"].(string); ok {
					errorMsg = msg
				}
			}
			return s.sendErrorAndEnd(c, errorMsg)
		}
	}
}

// processOpenAIStream processes the SSE stream from OpenAI Responses API
func (s *AccountTestService) processOpenAIStream(c *gin.Context, account *Account, body io.Reader) error {
	reader := bufio.NewReader(body)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return s.sendErrorAndEnd(c, "Stream ended before response.completed")
			}
			return s.sendErrorAndEnd(c, fmt.Sprintf("Stream read error: %s", err.Error()))
		}

		line = strings.TrimSpace(line)
		if line == "" || !sseDataPrefix.MatchString(line) {
			continue
		}

		jsonStr := sseDataPrefix.ReplaceAllString(line, "")
		if jsonStr == "[DONE]" {
			return s.sendErrorAndEnd(c, "Stream ended before response.completed")
		}

		var data map[string]any
		if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
			continue
		}

		eventType, _ := data["type"].(string)

		switch eventType {
		case "response.output_text.delta":
			// OpenAI Responses API uses "delta" field for text content
			if delta, ok := data["delta"].(string); ok && delta != "" {
				s.sendEvent(c, TestEvent{Type: "content", Text: delta})
			}
		case "response.completed", "response.done":
			return s.completeSuccessfulTest(c, account)
		case "response.failed":
			errorMsg := "OpenAI response failed"
			if responseData, ok := data["response"].(map[string]any); ok {
				if errData, ok := responseData["error"].(map[string]any); ok {
					if msg, ok := errData["message"].(string); ok && msg != "" {
						errorMsg = msg
					}
				}
			}
			return s.sendErrorAndEnd(c, errorMsg)
		case "error":
			errorMsg := "Unknown error"
			if errData, ok := data["error"].(map[string]any); ok {
				if msg, ok := errData["message"].(string); ok {
					errorMsg = msg
				}
			}
			return s.sendErrorAndEnd(c, errorMsg)
		}
	}
}

// processOpenAIChatCompletionsStream processes SSE stream from OpenAI-compatible
// chat/completions upstreams used by the CC passthrough test path.
func (s *AccountTestService) processOpenAIChatCompletionsStream(c *gin.Context, account *Account, body io.Reader) error {
	reader := bufio.NewReader(body)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return s.sendErrorAndEnd(c, "Stream ended before completion")
			}
			return s.sendErrorAndEnd(c, fmt.Sprintf("Stream read error: %s", err.Error()))
		}

		line = strings.TrimSpace(line)
		if line == "" || !sseDataPrefix.MatchString(line) {
			continue
		}

		jsonStr := sseDataPrefix.ReplaceAllString(line, "")
		if jsonStr == "[DONE]" {
			return s.completeSuccessfulTest(c, account)
		}

		var data map[string]any
		if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
			continue
		}

		if errData, ok := data["error"].(map[string]any); ok {
			errorMsg := "Unknown error"
			if msg, ok := errData["message"].(string); ok && strings.TrimSpace(msg) != "" {
				errorMsg = strings.TrimSpace(msg)
			}
			return s.sendErrorAndEnd(c, errorMsg)
		}

		choices, ok := data["choices"].([]any)
		if !ok {
			continue
		}
		for _, choiceRaw := range choices {
			choice, ok := choiceRaw.(map[string]any)
			if !ok {
				continue
			}
			if delta, ok := choice["delta"].(map[string]any); ok {
				if text, ok := delta["content"].(string); ok && text != "" {
					s.sendEvent(c, TestEvent{Type: "content", Text: text})
				}
			}
			if finishReason, ok := choice["finish_reason"].(string); ok && finishReason != "" {
				return s.completeSuccessfulTest(c, account)
			}
		}
	}
}

// testOpenAIImageAPIKey tests OpenAI image generation using an API Key account.
func (s *AccountTestService) testOpenAIImageAPIKey(c *gin.Context, ctx context.Context, account *Account, modelID, prompt string) error {
	authToken := account.GetOpenAIApiKey()
	if authToken == "" {
		return s.sendErrorAndEnd(c, "No API key available")
	}

	baseURL := account.GetOpenAIBaseURL()
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
	if err != nil {
		return s.sendErrorAndEnd(c, s.formatBaseURLError(err))
	}
	apiURL := buildOpenAIImagesURL(normalizedBaseURL, openAIImagesGenerationsEndpoint)

	// Set SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	s.sendEvent(c, TestEvent{Type: "test_start", Model: modelID})

	payload := map[string]any{
		"model":           modelID,
		"prompt":          prompt,
		"n":               1,
		"response_format": "b64_json",
	}
	payloadBytes, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return s.sendErrorAndEnd(c, "Failed to create request")
	}
	req = req.WithContext(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)

	resp, err := s.doTestRequestWithTLS(ctx, account, req, s.tlsFPProfileService.ResolveTLSProfile(account))
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Request failed: %s", err.Error()))
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Failed to read response: %s", err.Error()))
	}

	if resp.StatusCode != http.StatusOK {
		return s.sendErrorAndEnd(c, fmt.Sprintf("API returned %d: %s", resp.StatusCode, string(body)))
	}

	// Parse {"data": [{"b64_json": "...", "revised_prompt": "..."}]}
	var result struct {
		Data []struct {
			B64JSON       string `json:"b64_json"`
			RevisedPrompt string `json:"revised_prompt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Failed to parse response: %s", err.Error()))
	}

	if len(result.Data) == 0 {
		return s.sendErrorAndEnd(c, "No images returned from API")
	}

	for _, item := range result.Data {
		if item.RevisedPrompt != "" {
			s.sendEvent(c, TestEvent{Type: "content", Text: item.RevisedPrompt})
		}
		if item.B64JSON != "" {
			s.sendEvent(c, TestEvent{
				Type:     "image",
				ImageURL: "data:image/png;base64," + item.B64JSON,
				MimeType: "image/png",
			})
		}
	}

	return s.completeSuccessfulTest(c, account)
}

// testOpenAIImageOAuth tests OpenAI image generation using an OAuth account via Codex /responses API.
func (s *AccountTestService) testOpenAIImageOAuth(c *gin.Context, ctx context.Context, account *Account, modelID, prompt string) error {
	authToken := account.GetOpenAIAccessToken()
	if authToken == "" {
		return s.sendErrorAndEnd(c, "No access token available")
	}

	// Set SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	s.sendEvent(c, TestEvent{Type: "test_start", Model: modelID})
	s.sendEvent(c, TestEvent{Type: "content", Text: "Calling Codex /responses image tool...\n"})

	parsed := &OpenAIImagesRequest{
		Endpoint: openAIImagesGenerationsEndpoint,
		Model:    strings.TrimSpace(modelID),
		Prompt:   prompt,
	}
	applyOpenAIImagesDefaults(parsed)

	responsesBody, err := buildOpenAIImagesResponsesRequest(parsed, parsed.Model)
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Failed to build image request: %s", err.Error()))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chatgptCodexAPIURL, bytes.NewReader(responsesBody))
	if err != nil {
		return s.sendErrorAndEnd(c, "Failed to create request")
	}
	req = req.WithContext(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI))
	req.Host = "chatgpt.com"
	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("OpenAI-Beta", "responses=experimental")
	req.Header.Set("Originator", "codex_cli_rs")
	req.Header.Set("Version", codexCLIVersion)
	if customUA := strings.TrimSpace(account.GetOpenAIUserAgent()); customUA != "" {
		req.Header.Set("User-Agent", customUA)
	}
	if !openai.IsCodexCLIRequest(req.Header.Get("User-Agent")) {
		req.Header.Set("User-Agent", codexCLIUserAgent)
	}
	probeSessionID := uuid.NewString()
	req.Header.Set("Session_ID", probeSessionID)
	req.Header.Set("Conversation_ID", probeSessionID)
	if chatgptAccountID := strings.TrimSpace(account.GetChatGPTAccountID()); chatgptAccountID != "" {
		req.Header.Set("chatgpt-account-id", chatgptAccountID)
	}

	resp, err := s.doTestRequestWithTLS(ctx, account, req, s.tlsFPProfileService.ResolveTLSProfile(account))
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Responses API request failed: %s", err.Error()))
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		message := strings.TrimSpace(extractUpstreamErrorMessage(body))
		if message == "" {
			message = fmt.Sprintf("Responses API returned %d", resp.StatusCode)
		}
		return s.sendErrorAndEnd(c, message)
	}

	return s.processOpenAIImageStream(c, account, resp.Body)
}

func (s *AccountTestService) processOpenAIImageStream(c *gin.Context, account *Account, body io.Reader) error {
	reader := bufio.NewReader(body)
	flusher, _ := c.Writer.(http.Flusher)
	var (
		sseData     openAISSEDataAccumulator
		streamMeta  openAIResponsesImageResult
		pending     []openAIResponsesImageResult
		pendingSeen = make(map[string]struct{})
		emitted     = make(map[string]struct{})
		createdAt   int64
		completed   bool
	)
	type streamReadResult struct {
		line []byte
		err  error
	}
	readCh := make(chan streamReadResult, 1)
	go func() {
		for {
			line, err := reader.ReadBytes('\n')
			readCh <- streamReadResult{line: append([]byte(nil), line...), err: err}
			if err != nil {
				close(readCh)
				return
			}
		}
	}()
	var heartbeatTicker *time.Ticker
	if flusher != nil && openAIImageTestHeartbeatInterval > 0 {
		heartbeatTicker = time.NewTicker(openAIImageTestHeartbeatInterval)
		defer heartbeatTicker.Stop()
	}

	emitImage := func(item openAIResponsesImageResult) {
		if item.RevisedPrompt != "" {
			s.sendEvent(c, TestEvent{Type: "content", Text: item.RevisedPrompt})
		}
		mimeType := openAIImageOutputMIMEType(item.OutputFormat)
		s.sendEvent(c, TestEvent{
			Type:     "image",
			ImageURL: "data:" + mimeType + ";base64," + item.Result,
			MimeType: mimeType,
		})
	}

	processPayload := func(payload []byte) error {
		if !gjson.ValidBytes(payload) {
			return nil
		}
		if meta, eventCreatedAt, ok := extractOpenAIResponsesImageMetaFromLifecycleEvent(payload); ok {
			mergeOpenAIResponsesImageMeta(&streamMeta, meta)
			if eventCreatedAt > 0 {
				createdAt = eventCreatedAt
			}
		}

		switch gjson.GetBytes(payload, "type").String() {
		case "response.output_item.done":
			item, itemID, ok, err := extractOpenAIImageFromResponsesOutputItemDone(payload)
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}
			mergeOpenAIResponsesImageMeta(&item, streamMeta)
			key := openAIResponsesImageResultKey(itemID, item)
			if _, seen := emitted[key]; seen {
				return nil
			}
			if _, seen := pendingSeen[key]; seen {
				return nil
			}
			pendingSeen[key] = struct{}{}
			pending = append(pending, item)
			return nil
		case "response.failed":
			errorMsg := "OpenAI image response failed"
			if responseData, ok := gjson.GetBytes(payload, "response.error.message").Value().(string); ok && strings.TrimSpace(responseData) != "" {
				errorMsg = strings.TrimSpace(responseData)
			}
			return fmt.Errorf("%s", errorMsg)
		case "error":
			errorMsg := strings.TrimSpace(gjson.GetBytes(payload, "error.message").String())
			if errorMsg == "" {
				errorMsg = "Unknown error"
			}
			return fmt.Errorf("%s", errorMsg)
		case "response.completed":
			results, completedAt, _, firstMeta, err := extractOpenAIImagesFromResponsesCompleted(payload)
			if err != nil {
				return err
			}
			if completedAt > 0 {
				createdAt = completedAt
			}
			mergeOpenAIResponsesImageMeta(&streamMeta, firstMeta)

			finalResults := make([]openAIResponsesImageResult, 0, len(results)+len(pending))
			finalSeen := make(map[string]struct{})
			for _, item := range results {
				mergeOpenAIResponsesImageMeta(&item, streamMeta)
				appendOpenAIResponsesImageResultDedup(&finalResults, finalSeen, "", item)
			}
			for _, item := range pending {
				mergeOpenAIResponsesImageMeta(&item, streamMeta)
				appendOpenAIResponsesImageResultDedup(&finalResults, finalSeen, "", item)
			}

			if len(finalResults) == 0 {
				return fmt.Errorf("no images returned from responses API")
			}

			for _, item := range finalResults {
				key := openAIResponsesImageResultKey("", item)
				if _, seen := emitted[key]; seen {
					continue
				}
				emitted[key] = struct{}{}
				emitImage(item)
			}
			completed = true
			_ = createdAt
			return nil
		default:
			return nil
		}
	}

	for {
		select {
		case <-c.Request.Context().Done():
			return s.sendErrorAndEnd(c, fmt.Sprintf("Stream read error: %s", c.Request.Context().Err()))
		case <-func() <-chan time.Time {
			if heartbeatTicker != nil {
				return heartbeatTicker.C
			}
			return nil
		}():
			if err := s.sendSSEComment(c, "keepalive"); err != nil {
				return s.sendErrorAndEnd(c, fmt.Sprintf("Stream write error: %s", err.Error()))
			}
		case result, ok := <-readCh:
			if !ok {
				if completed {
					return s.completeSuccessfulTest(c, account)
				}
				return s.sendErrorAndEnd(c, "Stream ended before response.completed")
			}
			line, err := result.line, result.err
			if len(line) > 0 {
				var completedPayloads [][]byte
				sseData.AddLine(string(line), func(data []byte) {
					completedPayloads = append(completedPayloads, append([]byte(nil), data...))
				})
				for _, payload := range completedPayloads {
					if processErr := processPayload(payload); processErr != nil {
						return s.sendErrorAndEnd(c, processErr.Error())
					}
				}
				if completed {
					return s.completeSuccessfulTest(c, account)
				}
			}
			if err != nil {
				if err == io.EOF {
					var flushErr error
					sseData.Flush(func(data []byte) {
						if processErr := processPayload(data); processErr != nil && flushErr == nil {
							flushErr = processErr
						}
					})
					if flushErr != nil {
						return s.sendErrorAndEnd(c, flushErr.Error())
					}
					if completed {
						return s.completeSuccessfulTest(c, account)
					}
					return s.sendErrorAndEnd(c, "Stream ended before response.completed")
				}
				return s.sendErrorAndEnd(c, fmt.Sprintf("Stream read error: %s", err.Error()))
			}
		}
	}
}

func (s *AccountTestService) sendEvent(c *gin.Context, event TestEvent) {
	eventJSON, _ := json.Marshal(event)
	if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", eventJSON); err != nil {
		log.Printf("failed to write SSE event: %v", err)
		return
	}
	c.Writer.Flush()
}

func (s *AccountTestService) sendSSEComment(c *gin.Context, comment string) error {
	if _, err := fmt.Fprintf(c.Writer, ": %s\n\n", strings.TrimSpace(comment)); err != nil {
		return err
	}
	c.Writer.Flush()
	return nil
}

func (s *AccountTestService) completeSuccessfulTest(c *gin.Context, account *Account) error {
	if err := s.bindPlatformDefaultGroupOnTestSuccess(c.Request.Context(), account); err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Failed to bind default group: %s", err.Error()))
	}
	s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
	return nil
}

func (s *AccountTestService) bindPlatformDefaultGroupOnTestSuccess(ctx context.Context, account *Account) error {
	if s == nil || s.accountRepo == nil || s.groupRepo == nil || account == nil || account.ID <= 0 {
		return nil
	}

	latest, err := s.accountRepo.GetByID(ctx, account.ID)
	if err != nil {
		return fmt.Errorf("refresh account: %w", err)
	}
	if latest == nil || latest.Platform == "" || len(latest.GroupIDs) > 0 {
		return nil
	}

	defaultGroupName := latest.Platform + "-default"
	groups, err := s.groupRepo.ListActiveByPlatform(ctx, latest.Platform)
	if err != nil {
		return fmt.Errorf("list platform groups: %w", err)
	}

	var defaultGroupID int64
	for _, group := range groups {
		if group.Name == defaultGroupName {
			defaultGroupID = group.ID
			break
		}
	}
	if defaultGroupID <= 0 {
		return nil
	}

	if err := s.accountRepo.BindGroups(ctx, latest.ID, []int64{defaultGroupID}); err != nil {
		return fmt.Errorf("bind account to %s: %w", defaultGroupName, err)
	}

	account.GroupIDs = []int64{defaultGroupID}
	return nil
}

// sendErrorAndEnd sends an error event and ends the stream
func (s *AccountTestService) sendErrorAndEnd(c *gin.Context, errorMsg string) error {
	errorMsg = s.formatTestErrorMessage(errorMsg)
	log.Printf("Account test error: %s", errorMsg)
	s.sendEvent(c, TestEvent{Type: "error", Error: errorMsg})
	return fmt.Errorf("%s", errorMsg)
}

func (s *AccountTestService) sendHTTPErrorAndEnd(c *gin.Context, statusCode int, errorMsg string) error {
	errorMsg = s.formatTestErrorMessage(errorMsg)
	log.Printf("Account test error: %s", errorMsg)
	s.sendEvent(c, TestEvent{
		Type:  "error",
		Code:  strconv.Itoa(statusCode),
		Error: errorMsg,
	})
	return fmt.Errorf("%s", errorMsg)
}

// RunTestBackground executes an account test in-memory (no real HTTP client),
// capturing SSE output via httptest.NewRecorder, then parses the result.
func (s *AccountTestService) RunTestBackground(ctx context.Context, accountID int64, modelID string) (*ScheduledTestResult, error) {
	startedAt := time.Now()

	w := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(w)
	ginCtx.Request = (&http.Request{}).WithContext(ctx)

	testErr := s.TestAccountConnection(ginCtx, accountID, modelID, "", "default")

	finishedAt := time.Now()
	body := w.Body.String()
	responseText, errMsg, httpStatusCode := parseTestSSEOutput(body)

	status := "success"
	if testErr != nil || errMsg != "" {
		status = "failed"
		if errMsg == "" && testErr != nil {
			errMsg = testErr.Error()
		}
	}

	return &ScheduledTestResult{
		Status:         status,
		ResponseText:   responseText,
		ErrorMessage:   errMsg,
		HTTPStatusCode: httpStatusCode,
		LatencyMs:      finishedAt.Sub(startedAt).Milliseconds(),
		StartedAt:      startedAt,
		FinishedAt:     finishedAt,
	}, nil
}

// parseTestSSEOutput extracts response text and error message from captured SSE output.
func parseTestSSEOutput(body string) (responseText, errMsg string, httpStatusCode *int) {
	var texts []string
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		jsonStr := strings.TrimPrefix(line, "data: ")
		var event TestEvent
		if err := json.Unmarshal([]byte(jsonStr), &event); err != nil {
			continue
		}
		switch event.Type {
		case "content":
			if event.Text != "" {
				texts = append(texts, event.Text)
			}
		case "error":
			errMsg = event.Error
			if event.Code != "" {
				if parsed, err := strconv.Atoi(strings.TrimSpace(event.Code)); err == nil {
					code := parsed
					httpStatusCode = &code
				}
			}
		}
	}
	responseText = strings.Join(texts, "")
	return
}
