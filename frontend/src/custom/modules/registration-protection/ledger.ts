const registrationBalanceTypes = new Set(['signup_bonus'])

export function isRegistrationBalanceType(type: string): boolean {
  return registrationBalanceTypes.has(type)
}

export function registrationHistoryTypeKey(type: string): string | undefined {
  return registrationBalanceTypes.has(type) ? 'redeem.signupBonus' : undefined
}
