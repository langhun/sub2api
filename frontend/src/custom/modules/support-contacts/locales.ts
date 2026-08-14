export const supportContactsLocaleMessages = {
  en: {
    supportContacts: {
      openHint: 'View contact methods and QR codes', dialogHint: 'Choose a contact method, then copy the account or scan its QR code.', copy: 'Copy', open: 'Open link', copied: 'Copied to clipboard',
      channels: { wechat: 'WeChat', qq: 'QQ', telegram: 'Telegram', email: 'Email', link: 'Link' },
    },
    admin: { supportContacts: {
      title: 'Customer support contacts', description: 'Manage the contact methods, QR codes, and links shown to signed-in users.', empty: 'No customer support contacts configured.', saved: 'Saved contact', newContact: 'New contact', moveUp: 'Move up', moveDown: 'Move down', enabled: 'Visible to users', channel: 'Channel', name: 'Display name', contact: 'Contact information', contactPlaceholder: 'WeChat ID, email address, or account name', link: 'Open link', linkPlaceholder: 'https://example.com or mailto:help(at)example.com', linkHint: 'Optional. HTTPS, mailto, and tg links are supported.', qrCode: 'QR code', noQr: 'No QR code', imageUrl: 'Image URL', uploadImage: 'Upload image', qrImageUrl: 'QR image URL', selectImage: 'Select PNG, JPEG, or WebP', qrHint: 'Each contact can have one QR image, up to 2 MiB.', add: 'Add contact', savedSuccess: 'Customer support contacts saved', saveFailed: 'Failed to save customer support contacts', loadFailed: 'Failed to load customer support contacts', deleted: 'Customer support contact deleted', deleteFailed: 'Failed to delete customer support contact', invalidImage: 'Select a PNG, JPEG, or WebP image smaller than 2 MiB.', uploaded: 'QR image uploaded', uploadFailed: 'Failed to upload QR image',
    } },
  },
  zh: {
    supportContacts: {
      openHint: '查看联系方式和二维码', dialogHint: '选择联系方式后复制账号，或扫描对应二维码添加客服。', copy: '复制账号', open: '打开链接', copied: '已复制到剪贴板',
      channels: { wechat: '微信', qq: 'QQ', telegram: 'Telegram', email: '邮箱', link: '链接' },
    },
    admin: { supportContacts: {
      title: '客服资料', description: '管理已登录用户看到的多种客服联系方式、二维码和跳转链接。', empty: '暂未配置客服联系方式。', saved: '已保存', newContact: '新联系方式', moveUp: '上移', moveDown: '下移', enabled: '对用户显示', channel: '渠道', name: '显示名称', contact: '联系方式', contactPlaceholder: '微信号、邮箱或账号名称', link: '跳转链接', linkPlaceholder: 'https://example.com 或 mailto:help(at)example.com', linkHint: '可选，支持 HTTPS、mailto 和 tg 链接。', qrCode: '二维码', noQr: '不使用二维码', imageUrl: '图片链接', uploadImage: '上传图片', qrImageUrl: '二维码图片链接', selectImage: '选择 PNG、JPEG 或 WebP', qrHint: '每条联系方式最多一张二维码，大小不超过 2 MiB。', add: '添加联系方式', savedSuccess: '客服资料已保存', saveFailed: '保存客服资料失败', loadFailed: '加载客服资料失败', deleted: '客服联系方式已删除', deleteFailed: '删除客服联系方式失败', invalidImage: '请选择小于 2 MiB 的 PNG、JPEG 或 WebP 图片。', uploaded: '二维码已上传', uploadFailed: '二维码上传失败',
    } },
  },
} as const
