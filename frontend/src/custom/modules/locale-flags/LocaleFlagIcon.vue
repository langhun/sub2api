<template>
  <img
    v-if="flagSource"
    :src="flagSource"
    alt=""
    aria-hidden="true"
    width="20"
    height="15"
    class="h-[15px] w-5 shrink-0 rounded-[1px] object-cover"
    draggable="false"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import chinaFlag from './assets/cn.svg'
import unitedStatesFlag from './assets/us.svg'

const props = defineProps<{
  locale: string
}>()

const flagSources: Readonly<Record<string, string>> = {
  en: unitedStatesFlag,
  zh: chinaFlag,
}

const flagSource = computed(() => flagSources[props.locale] ?? '')
</script>
