export const supportTicketLocaleMessages = {
  en: {
    supportTickets: {
      title: 'Support tickets', description: 'Contact the service team and keep the conversation in one place.', newTicket: 'New ticket', subject: 'Subject', subjectPlaceholder: 'Briefly describe the issue', category: 'Category', message: 'Message', messagePlaceholder: 'Include the information needed to investigate the issue', create: 'Create ticket', createSuccess: 'Ticket created', reply: 'Reply', replyPlaceholder: 'Write a reply', send: 'Send reply', sendSuccess: 'Reply sent', reopen: 'Reopen ticket', reopenSuccess: 'Ticket reopened', loadFailed: 'Could not load support tickets. Try again.', empty: 'No support tickets yet', noMessages: 'No messages yet', backToList: 'All tickets', latestActivity: 'Last activity', status: 'Status', priority: 'Priority', ticketNumber: 'Ticket {number}', sensitiveNotice: 'Do not include API keys, passwords, verification codes, or other credentials.', categories: { account: 'Account and sign-in', billing: 'Billing and orders', api: 'API request issue', model: 'Model or channel issue', feature: 'Feature request', other: 'Other' }, statuses: { open: 'Waiting for support', waiting_user: 'Waiting for you', resolved: 'Resolved', closed: 'Closed' }, priorities: { low: 'Low', normal: 'Normal', high: 'High', urgent: 'Urgent' }, sender: { user: 'You', admin: 'Support', system: 'System' }, pagination: 'Showing {count} of {total}',
    },
  },
  zh: {
    supportTickets: {
      title: '工单支持', description: '与客服保持连续沟通，处理进度和历史回复都在这里。', newTicket: '新建工单', subject: '主题', subjectPlaceholder: '请简要说明遇到的问题', category: '问题分类', message: '问题描述', messagePlaceholder: '请提供排查所需的信息和复现步骤', create: '提交工单', createSuccess: '工单已创建', reply: '继续回复', replyPlaceholder: '请输入回复内容', send: '发送回复', sendSuccess: '回复已发送', reopen: '重新打开工单', reopenSuccess: '工单已重新打开', loadFailed: '工单加载失败，请重试。', empty: '还没有工单', noMessages: '暂无消息', backToList: '全部工单', latestActivity: '最后活动', status: '状态', priority: '优先级', ticketNumber: '工单 {number}', sensitiveNotice: '请勿提交 API Key、密码、验证码或其他凭证信息。', categories: { account: '账户与登录', billing: '充值与订单', api: 'API 调用异常', model: '模型或渠道问题', feature: '功能建议', other: '其他问题' }, statuses: { open: '等待客服处理', waiting_user: '等待您回复', resolved: '已解决', closed: '已关闭' }, priorities: { low: '低', normal: '普通', high: '高', urgent: '紧急' }, sender: { user: '您', admin: '客服', system: '系统' }, pagination: '显示 {count} / {total} 条',
    },
  },
} as const

export const supportTicketAdminLocaleMessages = {
  en: {
    supportTickets: {
      title: 'Support tickets', description: 'Review customer requests and maintain an auditable conversation.', allStatuses: 'All statuses', allCategories: 'All categories', allPriorities: 'All priorities', user: 'User', requester: 'Requester', internalNote: 'Internal note', internalNotePlaceholder: 'Only administrators can see this note', update: 'Update', updateSuccess: 'Ticket updated', details: 'Ticket details', replyToUser: 'Reply to user', noTickets: 'No tickets match these filters',
    },
  },
  zh: {
    supportTickets: {
      title: '工单管理', description: '集中处理客户请求，并保留可追溯的完整沟通记录。', allStatuses: '全部状态', allCategories: '全部分类', allPriorities: '全部优先级', user: '用户', requester: '提交人', internalNote: '内部备注', internalNotePlaceholder: '仅管理员可见，不会发送给用户', update: '更新工单', updateSuccess: '工单已更新', details: '工单信息', replyToUser: '回复用户', noTickets: '当前筛选条件下没有工单',
    },
  },
} as const
