package service

import "strings"

// knownForeignModelFamilyForModel returns the well-known upstream model
// family encoded in model. It intentionally recognizes only families whose
// protocol/provider ownership is unambiguous; unknown names remain eligible
// for OpenAI-compatible custom providers.
//
// Model IDs are not guaranteed to use one canonical spelling. In addition to
// the usual "gemini-*" / "claude-*" forms, providers commonly expose IDs such
// as "google/gemini-*", "models/gemini-*", "anthropic/claude-*", and
// compact aliases such as "gemini3-*" / "claude3-*". The classifier checks
// path/namespace segments while preserving boundaries so names like
// "geminix-proxy" and "claudette" are not misclassified.
func knownForeignModelFamilyForModel(model string) string {
	normalized := strings.ToLower(strings.TrimSpace(model))
	if normalized == "" {
		return ""
	}

	// Bedrock IDs carry the provider namespace before the model name. Keep the
	// explicit prefixes for compatibility with the existing dispatch guard,
	// including IDs whose model segment is not yet in our family table.
	for _, prefix := range []struct {
		prefix string
		family string
	}{
		{prefix: "anthropic.", family: "anthropic"},
		{prefix: "amazon.", family: "anthropic"},
		{prefix: "us.anthropic.", family: "anthropic"},
		{prefix: "global.anthropic.", family: "anthropic"},
	} {
		if strings.HasPrefix(normalized, prefix.prefix) {
			return prefix.family
		}
	}

	// Slash, dot, colon, and underscore are namespace/variant separators in
	// the model IDs seen in the gateway. Hyphens stay inside a segment so the
	// boundary checks below can distinguish gemini-* from geminix-*.
	segments := strings.FieldsFunc(normalized, func(r rune) bool {
		switch r {
		case '/', '.', ':', '_':
			return true
		default:
			return false
		}
	})
	for _, segment := range segments {
		switch {
		case modelFamilyPrefixMatch(segment, "gemini"):
			return "gemini"
		case modelFamilyPrefixMatch(segment, "claude"):
			return "anthropic"
		}
	}

	return ""
}

func modelFamilyPrefixMatch(segment, family string) bool {
	if segment == family {
		return true
	}
	if !strings.HasPrefix(segment, family) {
		return false
	}
	rest := segment[len(family):]
	if rest == "" {
		return true
	}
	// A hyphen or a numeric suffix covers the ordinary and compact forms
	// (gemini-2.5, gemini3, claude3-opus). Do not reject arbitrary aliases that
	// merely happen to begin with the family spelling (for example geminix).
	return rest[0] == '-' || (rest[0] >= '0' && rest[0] <= '9')
}

// SmartRoutingAccountModelCompatible rejects an account-level mapping that
// would send an OpenAI/GPT request to a known foreign provider family. It is
// intentionally scoped to smart-routing admission; ordinary account mapping
// behavior remains unchanged for legacy single-group requests.
func SmartRoutingAccountModelCompatible(account *Account, requestedModel string) bool {
	if account == nil || strings.TrimSpace(requestedModel) == "" {
		return account != nil
	}
	if account.Platform != PlatformOpenAI && account.Platform != PlatformGrok {
		return true
	}
	mapped, matched := account.ResolveMappedModel(requestedModel)
	if !matched {
		return true
	}
	return knownForeignModelFamilyForModel(mapped) == ""
}

// resolveOpenAIForwardModel 解析 OpenAI 兼容转发使用的模型。
// messagesDispatchMappedModel 是调用方已为 /v1/messages 解析的显式调度结果；
// 普通 OpenAI 请求必须传空，避免将分组配置作为通用模型兜底。
func resolveOpenAIForwardModel(account *Account, requestedModel, messagesDispatchMappedModel string) string {
	messagesDispatchMappedModel = strings.TrimSpace(messagesDispatchMappedModel)
	if account == nil {
		if messagesDispatchMappedModel != "" {
			return messagesDispatchMappedModel
		}
		return requestedModel
	}

	mappedModel, matched := account.ResolveMappedModel(requestedModel)
	if !matched && messagesDispatchMappedModel != "" {
		return messagesDispatchMappedModel
	}
	return mappedModel
}

// openAIOAuthForeignModelPrefixes 列出明确属于其他厂商家族的模型名前缀。
// Codex 上游不可能服务这些模型：转发阶段 normalizeOpenAIModelForUpstream
// 对未知模型原样透传，上游必然返回不可重试的 400。
//
// 采用保守黑名单而非 Codex 模型白名单：未知/自定义别名保持「允许」，
// 以兼容渠道级模型映射等「账号选定之后才改写模型名」的部署方式
// （调度过滤看到的是改写前的原始模型名）。前缀分类的先例见
// ResolveThinkingProtocol（thinking_protocol.go）。
var openAIOAuthForeignModelPrefixes = []string{
	"deepseek-",
	"glm-",
	"kimi-",
	"moonshot-",
	"qwen-",
	"qwen2-",
	"qwen3-",
	"qwen4-",
	"qwq-",
	"minimax-",
	"gemini-",
	"gemma-",
	"grok-",
	"doubao-",
	"hunyuan-",
	"llama-",
	"llama2-",
	"llama3-",
	"meta-llama",
	"mistral-",
	"mixtral-",
	"baichuan-",
	"ernie-",
	"step-",
	"seed-",
	"yi-",
}

// isOpenAIOAuthServableModel 判断「空 model_mapping 的 OpenAI OAuth 账号」能否
// 服务请求模型。空映射默认仍是「允许」，仅排除明确属于其他厂商家族的模型
// （deepseek-*/glm-*、以及 Kimi Code 官方 bare ID k3 / k3-256k 等）——这类
// 请求原样透传必然被 Codex 上游以不可重试的 400 拒绝，且不触发 failover，
// 应在调度阶段就跳过该账号，把请求让给显式声明支持该模型的账号（#3662）。
// bare k3 仅精确匹配（取 last segment 后），不使用宽泛 k3- 前缀，以免误伤
// 自定义别名；显式 model_mapping 命中路径不经过本函数，语义不变。
func isOpenAIOAuthServableModel(requestedModel string) bool {
	model := strings.ToLower(lastOpenAIModelSegment(requestedModel))
	if model == "" {
		return true // 空模型交由上层必填校验处理
	}
	// Kimi Code 官方 bare model ID：无厂商前缀，prefix 黑名单挡不住。
	if model == "k3" || model == "k3-256k" {
		return false
	}
	for _, prefix := range openAIOAuthForeignModelPrefixes {
		if strings.HasPrefix(model, prefix) {
			return false
		}
	}
	return true
}

// resolveOpenAICompactForwardModel determines the compact-only upstream model
// for /responses/compact requests. It never affects normal /responses traffic.
// When no compact-specific mapping matches, the input model is returned as-is.
func resolveOpenAICompactForwardModel(account *Account, model string) string {
	trimmedModel := strings.TrimSpace(model)
	if trimmedModel == "" || account == nil {
		return trimmedModel
	}

	mappedModel, matched := account.ResolveCompactMappedModel(trimmedModel)
	if !matched {
		return trimmedModel
	}
	if trimmedMapped := strings.TrimSpace(mappedModel); trimmedMapped != "" {
		return trimmedMapped
	}
	return trimmedModel
}
