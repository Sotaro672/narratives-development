// frontend/mall/src/features/landing/components/pricingPlans.ts

import type { SubscriptionPlanRow } from "./PricingPlanTable";

export const subscriptionPlanColumns = [
  "Starter",
  "Simple",
  "Grow",
  "Advanced",
  "Enterprise",
];

export const subscriptionPlanRows: SubscriptionPlanRow[] = [
  {
    label: "料金",
    values: [
      "4,990円/月",
      "9,990円/月",
      "19,990円/月",
      "29,990円/月",
      "別途相談",
    ],
  },
  {
    label: "電子名札発行",
    values: ["〇", "〇", "〇", "〇", "〇"],
  },
  {
    label: "お問い合わせ",
    values: ["〇", "〇", "〇", "〇", "〇"],
  },
  {
    label: "レビュー",
    values: ["×", "〇", "〇", "〇", "〇"],
  },
  {
    label: "一斉告知",
    values: ["×", "×", "〇", "〇", "〇"],
  },
  {
    label: "メンバー招待",
    values: ["×", "×", "×", "〇", "〇"],
  },
  {
    label: "ブランド数",
    values: ["1", "1", "1", "無制限", "無制限"],
  },
];