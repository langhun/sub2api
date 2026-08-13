export const openAIRTAdminLocaleMessages = {
  en: {
    openaiRt: {
      manage: 'Manage RT',
      title: 'OpenAI refresh token',
      oldToken: 'Current RT',
      newToken: 'New RT',
      newTokenPlaceholder: 'Paste a new refresh token',
      reveal: 'Show',
      hide: 'Hide',
      copy: 'Copy',
      copied: 'Copied',
      loading: 'Loading token...',
      save: 'Replace RT',
      replacing: 'Validating and replacing...',
      confirm: 'Replace this account RT? The current token will stop being used.',
      required: 'Enter a new RT first.',
      loadFailed: 'Unable to load the current RT.',
      replaceSuccess: 'RT replaced successfully.',
      replaceFailed: 'Unable to replace RT.',
      invalidToken: 'This RT is invalid or expired. Sign in again and provide a new RT.',
      copyFailed: 'Unable to copy RT to the clipboard.',
      securityHint: 'This token is sensitive. Do not share it or save it in screenshots.'
    }
  },
  zh: {
    openaiRt: {
      manage: '管理 RT',
      title: 'OpenAI 刷新令牌',
      oldToken: '当前 RT',
      newToken: '新 RT',
      newTokenPlaceholder: '粘贴新的刷新令牌',
      reveal: '显示',
      hide: '隐藏',
      copy: '复制',
      copied: '已复制',
      loading: '正在读取令牌...',
      save: '替换 RT',
      replacing: '正在验证并替换...',
      confirm: '确定替换此账号的 RT 吗？当前令牌将停止使用。',
      required: '请先输入新的 RT。',
      loadFailed: '无法读取当前 RT。',
      replaceSuccess: 'RT 已替换。',
      replaceFailed: '无法替换 RT。',
      invalidToken: '该 RT 已失效或过期，请重新登录获取新的 RT。',
      copyFailed: '无法复制 RT 到剪贴板。',
      securityHint: '这是敏感令牌，请勿分享，也不要保存到截图中。'
    }
  }
} as const
