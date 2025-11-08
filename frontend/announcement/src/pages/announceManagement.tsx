// frontend/announcement/src/pages/announceManagement.tsx
import * as React from "react";
import PageStyle from "../../../shell/src/layout/PageStyle/PageStyle";
import { MOCK_ANNOUNCES, type Announce } from "../../mockdata";

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
