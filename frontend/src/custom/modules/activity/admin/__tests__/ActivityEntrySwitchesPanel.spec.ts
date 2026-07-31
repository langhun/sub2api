import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import ActivityEntrySwitchesPanel from '../ActivityEntrySwitchesPanel.vue'
import { activityAdminLocaleMessages } from '../locales'

const initialSettings = {
  usage_query_enabled: true,
  leaderboard_enabled: true,
  leaderboard_balance_enabled: true,
  leaderboard_consumption_enabled: true,
  leaderboard_checkin_enabled: true,
  leaderboard_include_admin: false,
}

const UsagePanelStub = {
  props: ['modelValue'],
  emits: ['update:modelValue'],
  template: '<button data-testid="usage" @click="$emit(\'update:modelValue\', { usage_query_enabled: false })">usage</button>',
}

const LeaderboardPanelStub = {
  props: ['modelValue'],
  emits: ['update:modelValue'],
	template: '<button data-testid="leaderboard" @click="$emit(\'update:modelValue\', { ...modelValue, leaderboard_checkin_enabled: false })">leaderboard</button>',
}

function mountPanel() {
  return mount(ActivityEntrySwitchesPanel, {
    props: { modelValue: initialSettings },
    global: {
      plugins: [createI18n({
        legacy: false,
        locale: 'en',
        missingWarn: false,
        fallbackWarn: false,
        messages: { en: {} },
      })],
      stubs: {
        ActivityUsageSettingsPanel: UsagePanelStub,
        ActivityLeaderboardSettingsPanel: LeaderboardPanelStub,
      },
    },
  })
}

describe('ActivityEntrySwitchesPanel', () => {
  it('owns the activity-specific card title and description', () => {
    const wrapper = mountPanel()

    expect(activityAdminLocaleMessages.en.settings.balanceFeatures).toMatchObject({
      entrySwitchesTitle: 'Entry and activity switches',
      entrySwitchesDescription: 'Control user entries, leaderboard tabs, and API access.',
    })
    expect(wrapper.find('.card').exists()).toBe(true)
  })

  it('combines child panel updates into one settings payload', async () => {
    const wrapper = mountPanel()

    await wrapper.get('[data-testid="usage"]').trigger('click')
    await wrapper.get('[data-testid="leaderboard"]').trigger('click')

    expect(wrapper.emitted('update:modelValue')).toEqual([
      [{ ...initialSettings, usage_query_enabled: false }],
		[{ ...initialSettings, usage_query_enabled: false, leaderboard_checkin_enabled: false }],
    ])
  })
})
