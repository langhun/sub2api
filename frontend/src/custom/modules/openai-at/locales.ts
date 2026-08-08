export const openAIATAdminLocaleMessages = {
  en: {
    admin: {
      openaiAt: {
        manage: 'Manage AT',
        title: 'OpenAI personal access token',
        oldToken: 'Current AT',
        newToken: 'New AT',
        newTokenPlaceholder: 'Paste a new at-... token',
        reveal: 'Show',
        hide: 'Hide',
        copy: 'Copy',
        copied: 'Copied',
        loading: 'Loading token...',
        save: 'Replace AT',
        replacing: 'Validating and replacing...',
        confirm: 'Replace this account AT? The current token will stop being used.',
        required: 'Enter a new AT first.',
        loadFailed: 'Unable to load the current AT.',
        replaceSuccess: 'AT replaced successfully.',
        replaceFailed: 'Unable to replace AT.',
        copyFailed: 'Unable to copy AT to the clipboard.',
        securityHint: 'This token is sensitive. Do not share it or save it in screenshots.'
      }
    }
  },
  zh: {
    admin: {
      openaiAt: {
        manage: '管理 AT',
        title: 'OpenAI 个人访问令牌',
        oldToken: '当前 AT',
        newToken: '新 AT',
        newTokenPlaceholder: '粘贴新的 at-... 令牌',
        reveal: '显示',
        hide: '隐藏',
        copy: '复制',
        copied: '已复制',
        loading: '正在读取令牌...',
        save: '替换 AT',
        replacing: '正在验证并替换...',
        confirm: '确定替换此账号的 AT 吗？当前令牌将停止使用。',
        required: '请先输入新的 AT。',
        loadFailed: '无法读取当前 AT。',
        replaceSuccess: 'AT 已替换。',
        replaceFailed: '无法替换 AT。',
        copyFailed: '无法复制 AT 到剪贴板。',
        securityHint: '这是敏感令牌，请勿分享，也不要保存到截图中。'
      }
    }
  }
} as const
