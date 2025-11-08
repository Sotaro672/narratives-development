// frontend/announce/src/pages/announceManagement.tsx
import * as React from "react";
import PageStyle from "../../../shell/src/layout/PageStyle/PageStyle";

type Announce = {
  id: string;
  title: string;
  body: string;
  category: "システム" | "メンテナンス" | "アップデート" | "お知らせ";
  publishedAt: string; // YYYY/MM/DD
  status: "公開中" | "下書き";
};

const MOCK_ANNOUNCES: Announce[] = [
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

export default function AnnounceManagementPage() {
  const [rows] = React.useState<Announce[]>(MOCK_ANNOUNCES);

  return (
    <PageStyle
      layout="single"
      title="お知らせ管理"
      // 管理トップ想定のため戻るボタンなし（必要なら onBack を渡してください）
    >
      <div className="announce-list">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b text-[0.75rem] text-muted-foreground">
              <th className="text-left py-2 px-2 w-32">ID</th>
              <th className="text-left py-2 px-2">件名</th>
              <th className="text-left py-2 px-2 w-32">カテゴリ</th>
              <th className="text-left py-2 px-2 w-32">状態</th>
              <th className="text-left py-2 px-2 w-40">公開日</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((a) => (
              <tr key={a.id} className="border-b last:border-b-0">
                <td className="py-2 px-2 text-xs text-blue-600">{a.id}</td>
                <td className="py-2 px-2">
                  <div className="font-medium">{a.title}</div>
                  <div className="text-xs text-muted-foreground line-clamp-1">
                    {a.body}
                  </div>
                </td>
                <td className="py-2 px-2 text-xs">
                  <span className="inline-flex items-center px-2 py-1 rounded-full bg-slate-100 text-slate-700">
                    {a.category}
                  </span>
                </td>
                <td className="py-2 px-2 text-xs">
                  {a.status === "公開中" ? (
                    <span className="inline-flex items-center px-2 py-1 rounded-full bg-emerald-50 text-emerald-700">
                      公開中
                    </span>
                  ) : (
                    <span className="inline-flex items-center px-2 py-1 rounded-full bg-slate-50 text-slate-500">
                      下書き
                    </span>
                  )}
                </td>
                <td className="py-2 px-2 text-xs">{a.publishedAt}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </PageStyle>
  );
}
