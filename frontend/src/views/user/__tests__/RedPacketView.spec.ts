import { beforeEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";

import RedPacketView from "../RedPacketView.vue";

const { getMyRedPackets, getRedPacketDetail, createRedPacket, claimRedPacket } =
  vi.hoisted(() => ({
    getMyRedPackets: vi.fn(),
    getRedPacketDetail: vi.fn(),
    createRedPacket: vi.fn(),
    claimRedPacket: vi.fn(),
  }));

vi.mock("@/api/transfer", () => ({
  getMyRedPackets,
  getRedPacketDetail,
  createRedPacket,
  claimRedPacket,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: () => ({
    showSuccess: vi.fn(),
    showError: vi.fn(),
    showWarning: vi.fn(),
    showInfo: vi.fn(),
  }),
}));

vi.mock("@/stores/auth", () => ({
  useAuthStore: () => ({
    user: { balance: 100 },
    refreshUser: vi.fn().mockResolvedValue(undefined),
  }),
}));

vi.mock("vue-i18n", async () => {
  const actual = await vi.importActual<typeof import("vue-i18n")>("vue-i18n");
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, fallback?: string) => fallback ?? key,
    }),
  };
});

vi.mock("@/components/layout/AppLayout.vue", () => ({
  default: {
    template: "<div><slot /></div>",
  },
}));

vi.mock("@/components/icons/Icon.vue", () => ({
  default: {
    template: "<span />",
  },
}));

describe("RedPacketView claim detail display name", () => {
  beforeEach(() => {
    getMyRedPackets.mockReset();
    getRedPacketDetail.mockReset();
    createRedPacket.mockReset();
    claimRedPacket.mockReset();

    getMyRedPackets.mockResolvedValue({
      items: [
        {
          id: 7,
          sender_id: 1,
          total_amount: 100,
          total_count: 2,
          remaining_amount: 0,
          remaining_count: 0,
          redpacket_type: "equal",
          fee: 0,
          fee_rate: 0,
          code: "RP-TEST",
          status: "exhausted",
          memo: null,
          expire_at: "2026-06-02T15:36:00Z",
          created_at: "2026-06-02T15:00:00Z",
        },
      ],
      total: 1,
      page: 1,
      page_size: 10,
    });

    getRedPacketDetail.mockResolvedValue({
      redpacket: { id: 7 },
      claims: [
        {
          id: 1,
          redpacket_id: 7,
          user_id: 2,
          user_email: "fallback@example.com",
          user_display_name: "named-user",
          amount: 10,
          transfer_id: 11,
          created_at: "2026-06-02T15:36:48Z",
        },
        {
          id: 2,
          redpacket_id: 7,
          user_id: 3,
          user_email: "email-only@example.com",
          user_display_name: "email-only@example.com",
          amount: 20,
          transfer_id: 12,
          created_at: "2026-06-02T15:38:11Z",
        },
      ],
    });
  });

  it("shows username first and falls back to email in claim detail rows", async () => {
    const wrapper = mount(RedPacketView);

    await flushPromises();

    const detailToggle = wrapper.findAll("button").find((node) =>
      node.attributes("class")?.includes("h-7 w-7"),
    );
    expect(detailToggle).toBeDefined();

    await detailToggle!.trigger("click");
    await flushPromises();

    const text = wrapper.text();
    expect(text).toContain("named-user");
    expect(text).toContain("email-only@example.com");
    expect(text).not.toContain("fallback@example.com");
  });
});
