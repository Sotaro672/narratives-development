// frontend/announcement/mockdata.tsx
export type Announce = {
  id: string;
  title: string;
  body: string;
  category: "システム" | "メンテナンス" | "アップデート" | "お知らせ";
  publishedAt: string; // YYYY/MM/DD
  status: "公開中" | "下書き";
};

export const MOCK_ANNOUNCES: Announce[] = [
  {
    id: "ann_001",
    title: "Solid State Console ベータ版リリース",
    body: "一部機能をベータ版として公開しました。フィードバックをお寄せください。",
    category: "お知らせ",
    publishedAt: "2025/11/01",
    status: "公開中",
  },
  {
    id: "ann_002",
    title: "システムメンテナンスのお知らせ",
    body: "2025/11/10 01:00 - 03:00 の間、システムメンテナンスを実施します。",
    category: "メンテナンス",
    publishedAt: "2025/10/28",
    status: "公開中",
  },
];
