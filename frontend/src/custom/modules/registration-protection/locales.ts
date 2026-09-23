import { registrationRiskAdminLocaleMessages } from './admin-locale'

export const registrationProtectionLocaleMessages = {
  en: { ...registrationRiskAdminLocaleMessages.en, redeem: { signupBonus: 'Registration bonus' } },
  zh: { ...registrationRiskAdminLocaleMessages.zh, redeem: { signupBonus: '注册赠送' } },
} as const
