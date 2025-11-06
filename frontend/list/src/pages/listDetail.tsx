// frontend/list/src/pages/listDetail.tsx
import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../admin/src/pages/AdminCard";
import { Card, CardHeader, CardTitle, CardContent } from "../../../shared/ui/card";

export default function ListDetail() {
  const navigate = useNavigate();
  const { listId } = useParams<{ listId: string }>();

  // ─────────────────────────────────────────
  // モックデータ（商品リスト詳細）
  // ─────────────────────────────────────────
  const [listName] = React.useState("2025 春夏コレクション");
  const [brand] = React.useState("LUMINA Fashion");
  const [category] = React.useState("トップス");
  const [itemCount] = React.useState(24);
  const [status] = React.useState("公開中");
  const [updatedAt] = React.useState("2025/11/06 22:00");

  // ─────────────────────────────────────────
  // 管理情報（右カラム）
  // ─────────────────────────────────────────
  const [assignee, setAssignee] = React.useState("佐藤 美咲");
  const [creator] = React.useState("山田 太郎");
  const [createdAt] = React.useState("2025/10/25 14:30");

  // ─────────────────────────────────────────
  // 戻る
  // ─────────────────────────────────────────
  const onBack = React.useCallback(() => navigate(-1), [navigate]);

  return (
    <PageStyle
      layout="grid-2"
      title={`リスト詳細：${listId ?? "不明ID"}`}
      onBack={onBack}
      onSave={undefined}
    >
      {/* 左カラム：リスト詳細 */}
      <div>
        <Card>
          <CardHeader>
            <CardTitle>リスト情報</CardTitle>
          </CardHeader>
          <CardContent>
            <table className="w-full text-sm">
              <tbody>
                <tr>
                  <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                    リスト名
                  </th>
                  <td className="py-2">{listName}</td>
                </tr>
                <tr>
                  <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                    ブランド
                  </th>
                  <td className="py-2">{brand}</td>
                </tr>
                <tr>
                  <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                    カテゴリ
                  </th>
                  <td className="py-2">{category}</td>
                </tr>
                <tr>
                  <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                    商品数
                  </th>
                  <td className="py-2">{itemCount}</td>
                </tr>
                <tr>
                  <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                    ステータス
                  </th>
                  <td className="py-2">
                    <span
                      className="inline-flex items-center px-2 py-1 rounded-full text-xs font-semibold"
                      style={{
                        background:
                          status === "公開中"
                            ? "hsl(var(--muted))"
                            : "hsl(var(--destructive))",
                        color:
                          status === "公開中"
                            ? "hsl(var(--muted-foreground))"
                            : "hsl(var(--destructive-foreground))",
                      }}
                    >
                      {status}
                    </span>
                  </td>
                </tr>
                <tr>
                  <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                    更新日時
                  </th>
                  <td className="py-2">{updatedAt}</td>
                </tr>
              </tbody>
            </table>
          </CardContent>
        </Card>
      </div>

      {/* 右カラム：管理情報 */}
      <AdminCard
        title="管理情報"
        assigneeName={assignee}
        createdByName={creator}
        createdAt={createdAt}
        onEditAssignee={() => setAssignee("新担当者")}
        onClickAssignee={() => console.log("assignee clicked:", assignee)}
        onClickCreatedBy={() => console.log("createdBy clicked:", creator)}
      />
    </PageStyle>
  );
}
