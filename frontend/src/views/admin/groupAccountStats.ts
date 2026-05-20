import type { AdminGroup } from "@/types";

type GroupAccountStats = Pick<
  AdminGroup,
  "active_account_count" | "rate_limited_account_count"
>;

export const getAvailableAccountCount = (
  row: GroupAccountStats,
): number => Math.max(0, row.active_account_count ?? 0);
