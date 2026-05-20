import type { InjectionKey, Ref } from 'vue'

export const accountStatusNowMsKey: InjectionKey<Ref<number>> = Symbol('accountStatusNowMs')
