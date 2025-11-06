// frontend/operation/src/pages/tokenOperationDetail.tsx
import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../admin/src/pages/AdminCard";
import { Card, CardHeader, CardTitle, CardContent } from "../../../shared/ui/card";

type OperationType = "発行" | "配布" | "回収" | "凍結" | "解除";

export default function TokenOperationDetail() {
  const navigate = useNavigate();
  const { tokenOperationId } = useParams<{ tokenOperationId: string }>();

  // ─────────────────────────────────────────
  // モックデータ（表示用）
  // ─────────────────────────────────────────
  const [operationType] = React.useState<OperationType>("配布");
  const [tokenName] = React.useState("LUMINA VIP Token");
  const [symbol] = React.useState("LVIP");
  const [network] = React.useState("Solana");
  const [amount] = React.useState<number>(1_000);
  const [status] = React.useState<"Pending" | "Succeeded" | "Failed">("Succeeded");
  const [txSignature] = React.useState("5V6y...9kPQ");
  const [executedAt] = React.useState("2025/11/06 21:10");

  // 管理情報（右カラム）
  const [assignee, setAssignee] = React.useState("佐藤 美咲");
  const [creator] = React.useState("山田 太郎");
  const [createdAt] = React.useState("2025/11/06 20:55");

  // 戻る
  const onBack = React.useCallback(() => navigate(-1), [navigate]);

  return (
    <PageStyle
      layout="grid-2"
      title={`トークン運用：${tokenOperationId ?? "不明ID"}`}
      onBack={onBack}
      onSave={undefined}
    >
      {/* 左カラム：オペレーション詳細 */}
      <div>
        <Card>
          <CardHeader>
            <CardTitle>オペレーション詳細</CardTitle>
          </CardHeader>
          <CardContent>
            <table className="w-full text-sm">
              <tbody>
                <tr>
                  <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">区分</th>
                  <td className="py-2">{operationType}</td>
                </tr>
                <tr>
                  <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">トークン</th>
                  <td className="py-2">
                    {tokenName} <span className="text-muted-foreground">({symbol})</span>
                  </td>
                </tr>
                <tr>
                  <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">ネットワーク</th>
                  <td className="py-2">{network}</td>
                </tr>
                <tr>
                  <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">数量</th>
                  <td className="py-2">{amount.toLocaleString()}</td>
                </tr>
                <tr>
                  <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">ステータス</th>
                  <td className="py-2">
                    <span
                      className="inline-flex items-center px-2 py-1 rounded-full text-xs font-semibold"
                      style={{
                        background: status === "Succeeded" ? "hsl(var(--muted))" : "hsl(var(--destructive))",
                        color: status === "Succeeded" ? "hsl(var(--muted-foreground))" : "hsl(var(--destructive-foreground))",
                      }}
                    >
                      {status}
                    </span>
                  </td>
                </tr>
                <tr>
                  <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">Tx Signature</th>
                  <td className="py-2">
                    <code className="text-xs">{txSignature}</code>
                  </td>
                </tr>
                <tr>
                  <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">実行日時</th>
                  <td className="py-2">{executedAt}</td>
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
