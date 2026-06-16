import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import { describe, expect, it } from "vitest";

const currentDir = dirname(fileURLToPath(import.meta.url));
const appLayoutSource = readFileSync(resolve(currentDir, "../AppLayout.vue"), "utf8");
const accountTableFiltersSource = readFileSync(
  resolve(currentDir, "../../admin/account/AccountTableFilters.vue"),
  "utf8",
);
const totpSetupModalSource = readFileSync(
  resolve(currentDir, "../../user/profile/TotpSetupModal.vue"),
  "utf8",
);
const settingsViewSource = readFileSync(
  resolve(currentDir, "../../../views/admin/SettingsView.vue"),
  "utf8",
);
const tablePageLayoutSource = readFileSync(
  resolve(currentDir, "../TablePageLayout.vue"),
  "utf8",
);
const dataTableSource = readFileSync(
  resolve(currentDir, "../../common/DataTable.vue"),
  "utf8",
);
const styleSource = readFileSync(resolve(currentDir, "../../../style.css"), "utf8");

describe("dark theme surface regressions", () => {
  it("keeps the global mesh background subdued in dark mode", () => {
    expect(appLayoutSource).toContain("bg-mesh-gradient opacity-70 dark:opacity-10");
  });

  it("renders the account filter shell with a dedicated dark surface", () => {
    expect(accountTableFiltersSource).toContain("dark:border-dark-700/80");
    expect(accountTableFiltersSource).toContain(
      "dark:bg-[linear-gradient(135deg,rgba(15,23,42,0.94),rgba(17,24,39,0.98))]",
    );
    expect(accountTableFiltersSource).toContain("dark:shadow-none");
  });

  it("does not fall back to a white TOTP QR panel in dark mode", () => {
    expect(totpSetupModalSource).toContain("dark:bg-dark-900");
    expect(totpSetupModalSource).not.toContain("dark:bg-white");
  });

  it("keeps the settings tab shell on a dark panel background", () => {
    expect(settingsViewSource).toContain("dark:border-dark-700/80");
    expect(settingsViewSource).toContain("dark:bg-dark-800/95");
  });

  it("keeps mobile table pages on a dark surface instead of a transparent shell", () => {
    expect(tablePageLayoutSource).toContain("dark:border-dark-700/70");
    expect(tablePageLayoutSource).toContain("dark:bg-dark-900/80");
  });

  it("gives mobile data cards their own dark list surface", () => {
    expect(dataTableSource).toContain("rounded-2xl bg-gray-50/80 p-1.5 dark:bg-dark-900/80");
  });

  it("uses a solid dark card surface instead of a washed out transparent panel", () => {
    expect(styleSource).toContain("@apply bg-white dark:bg-dark-800;");
  });
});
