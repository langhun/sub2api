import { describe, expect, it } from "vitest";

import { getAvailableAccountCount } from "../groupAccountStats";

describe("groupAccountStats", () => {
  it("uses the backend active count directly for available accounts", () => {
    expect(
      getAvailableAccountCount({
        active_account_count: 2,
      }),
    ).toBe(2);
  });

  it("does not subtract temporarily limited accounts a second time", () => {
    expect(
      getAvailableAccountCount({
        active_account_count: 2,
        rate_limited_account_count: 5,
      }),
    ).toBe(2);
  });

  it("falls back to zero when the backend count is missing", () => {
    expect(getAvailableAccountCount({})).toBe(0);
  });
});
