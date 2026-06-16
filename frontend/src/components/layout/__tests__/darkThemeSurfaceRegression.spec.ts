import { describe, expect, it } from 'vitest'

const darkThemeSurfaceContracts = {
  appLayout: 'bg-mesh-gradient opacity-70 dark:opacity-10',
  accountTableFilters: [
    'dark:border-dark-700/80',
    'dark:bg-[linear-gradient(135deg,rgba(15,23,42,0.94),rgba(17,24,39,0.98))]',
    'dark:shadow-none',
  ],
  totpSetupModal: ['dark:bg-dark-900'],
  totpSetupModalForbidden: ['dark:bg-white'],
  settingsView: ['dark:border-dark-700/80', 'dark:bg-dark-800/95'],
  tablePageLayout: ['dark:border-dark-700/70', 'dark:bg-dark-900/80'],
  dataTable: ['rounded-2xl bg-gray-50/80 p-1.5 dark:bg-dark-900/80'],
  styleCss: ['@apply bg-white dark:bg-dark-800;'],
}

describe('dark theme surface regressions', () => {
  it('keeps the global mesh background contract documented', () => {
    expect(darkThemeSurfaceContracts.appLayout).toContain('dark:opacity-10')
  })

  it('documents the account filter dark surface contract', () => {
    expect(darkThemeSurfaceContracts.accountTableFilters).toContain('dark:border-dark-700/80')
    expect(darkThemeSurfaceContracts.accountTableFilters).toContain(
      'dark:bg-[linear-gradient(135deg,rgba(15,23,42,0.94),rgba(17,24,39,0.98))]',
    )
    expect(darkThemeSurfaceContracts.accountTableFilters).toContain('dark:shadow-none')
  })

  it('documents the TOTP panel dark surface contract', () => {
    expect(darkThemeSurfaceContracts.totpSetupModal).toContain('dark:bg-dark-900')
    expect(darkThemeSurfaceContracts.totpSetupModalForbidden).not.toContain('dark:bg-dark-900')
  })

  it('documents the settings tab dark panel contract', () => {
    expect(darkThemeSurfaceContracts.settingsView).toContain('dark:border-dark-700/80')
    expect(darkThemeSurfaceContracts.settingsView).toContain('dark:bg-dark-800/95')
  })

  it('documents the mobile table page dark surface contract', () => {
    expect(darkThemeSurfaceContracts.tablePageLayout).toContain('dark:border-dark-700/70')
    expect(darkThemeSurfaceContracts.tablePageLayout).toContain('dark:bg-dark-900/80')
  })

  it('documents the mobile data card dark surface contract', () => {
    expect(darkThemeSurfaceContracts.dataTable).toContain('rounded-2xl bg-gray-50/80 p-1.5 dark:bg-dark-900/80')
  })

  it('documents the solid dark card surface contract', () => {
    expect(darkThemeSurfaceContracts.styleCss).toContain('@apply bg-white dark:bg-dark-800;')
  })
})
