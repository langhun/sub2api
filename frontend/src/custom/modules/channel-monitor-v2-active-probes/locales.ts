export const channelMonitorV2ActiveProbeLocaleMessages = {
  en: {
    channelMonitorV2: { admin: { descriptionV2: 'V2 passive monitoring is active. Optional V1 probe targets can run alongside it and feed availability into this page.', tabV1History: 'V1 active probe targets' } },
    channelMonitorV2ActiveProbes: {
      settings: {
        title: 'V1 active probes in V2',
        hint: 'Reuse enabled V1 probe and quota + probe targets. Endpoints, keys, models, intervals, and jitter remain managed in the V1 target list.',
        on: 'Enabled', off: 'Disabled', manageTargets: 'Manage active probe targets',
        loadFailed: 'Failed to load active probe settings', saved: 'Active probe setting saved', saveFailed: 'Failed to save active probe setting',
      },
      source: { active: 'Active probe', passive: 'User traffic', checkedAt: 'Probed {time}' },
      display: {
        latencyHeader: 'Latency P50',
        responseLatency: 'Response time P50',
        latencyValue: 'Latency {value}',
        responseLatencyValue: 'Response time {value}',
        latencyDataset: 'Latency P50',
        latencyAxis: 'Latency',
        unavailable: 'Not sampled',
        cacheUnavailable: '—',
        cacheDetail: 'Active probes do not report cache usage; real traffic will populate this automatically.',
        cacheDetailUser: 'Cache hit rate will appear after real user traffic is sampled.',
        latencyDetail: 'Active probes measure complete response time; real traffic provides first-token latency.',
      },
    },
  },
  zh: {
    channelMonitorV2: { admin: { descriptionV2: '当前为 V2 被动监控；可同时启用 V1 主动探测目标，并把可用性结果展示在本页面。', tabV1History: 'V1 主动探测目标' } },
    channelMonitorV2ActiveProbes: {
      settings: {
        title: '在 V2 中启用 V1 主动探测',
        hint: '复用已启用的 V1「探活」和「配额 + 探活」目标。Endpoint、Key、模型、间隔和抖动仍在 V1 目标列表维护。',
        on: '已开启', off: '已关闭', manageTargets: '管理主动探测目标',
        loadFailed: '加载主动探测设置失败', saved: '主动探测设置已保存', saveFailed: '保存主动探测设置失败',
      },
      source: { active: '主动探测', passive: '用户流量', checkedAt: '探测于 {time}' },
      display: {
        latencyHeader: '延迟 P50',
        responseLatency: '响应时间 P50',
        latencyValue: '延迟 {value}',
        responseLatencyValue: '响应时间 {value}',
        latencyDataset: '延迟 P50',
        latencyAxis: '延迟',
        unavailable: '未采样',
        cacheUnavailable: '—',
        cacheDetail: '主动探测不会返回缓存用量；有真实用户请求后会自动显示。',
        cacheDetailUser: '产生真实用户流量后会显示缓存命中率。',
        latencyDetail: '主动探测测量完整响应时间；真实用户流量会提供首 Token 延迟。',
      },
    },
  },
} as const
