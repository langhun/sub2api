<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl space-y-4">
      <RouterLink to="/games" class="btn btn-secondary">返回娱乐大厅</RouterLink>
      <SlotMachinePanel />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import AppLayout from '@/components/layout/AppLayout.vue'
import SlotMachinePanel from '@/components/user/games/SlotMachinePanel.vue'
</script>
