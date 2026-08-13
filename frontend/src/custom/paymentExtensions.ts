import type { CallbackPaths, ConfigFieldDef, TypeOption } from '@/components/payment/providerConfig'

export interface CustomPaymentDefinition {
  providerKey: string
  providerLabelKey: string
  supportedTypes: readonly string[]
  paymentTypes: readonly TypeOption[]
  methodOrder?: number
  configFields: readonly ConfigFieldDef[]
  callbackPaths?: CallbackPaths
  webhookHintKey?: string
  methodIcon?: string
  selectedClass?: string
  buttonClass?: string
}

const definitions = new Map<string, CustomPaymentDefinition>()

export function registerCustomPaymentDefinition(definition: CustomPaymentDefinition): void {
  const key = definition.providerKey.trim()
  if (!key) throw new Error('custom payment provider key is required')
  definitions.set(key, { ...definition, providerKey: key })
}

export function customPaymentDefinitions(): readonly CustomPaymentDefinition[] {
  return [...definitions.values()]
}

export function customPaymentDefinition(key: string): CustomPaymentDefinition | undefined {
  return definitions.get(key.trim())
}

export function customPaymentTypeDefinition(type: string): CustomPaymentDefinition | undefined {
  const normalized = type.trim()
  return customPaymentDefinitions().find(definition =>
    definition.paymentTypes.some(option => option.value === normalized),
  )
}

export function customPaymentTypeOptions(): TypeOption[] {
  return customPaymentDefinitions().flatMap(definition => definition.paymentTypes.map(option => ({ ...option })))
}

export function customProviderKeyOptions(): TypeOption[] {
  return customPaymentDefinitions().map(definition => ({
    value: definition.providerKey,
    label: definition.providerLabelKey,
  }))
}
