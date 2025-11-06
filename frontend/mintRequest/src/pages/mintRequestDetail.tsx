// frontend/mintRequest/src/pages/mintRequestDetail.tsx
import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageHeader from "../../../shell/src/layout/PageHeader/PageHeader";
import { Card, CardHeader, CardTitle, CardContent } from "../../../shared/ui/card";

export default function MintRequestDetail() {
  const navigate = useNavigate();
  const { requestId } = useParams<{ requestId: string }>();

  // ─────────────────────────────────────────
  // モックデータ（ミント申請詳細）
  // ─────────────────────────────────────────
  const [tokenName] = React.useState("LUMINA VIP Token");
  const [brand] = React.useState("LUMINA Fashion");
  const [symbol] = React.useState("LVIP");
  const [requestedBy] = React.useState("山田 太郎");
  const [requestDate] = React.useState("2025/11/05 14:30");
  const [quantity] = React.useState(1000);
  const [status] = React.useState<"承認待ち" | "承認済み" | "却下">("承認待ち");
  const [remarks] = React.useState("VIP会員向け初回発行分として申請。");

  // ─────────────────────────────────────────
  // 戻るボタン
  // ─────────────────────────────────────────
  const onBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  return (
    <div className="p-6">
      <PageHeader title={`ミント申請詳細：${requestId ?? "不明ID"}`} onBack={onBack} />

      <Card className="mt-4">
        <CardHeader>
          <CardTitle>申請情報</CardTitle>
        </CardHeader>
        <CardContent>
          <table className="w-full text-sm">
            <tbody>
              <tr>
                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                  トークン名
                </th>
                <td className="py-2">{tokenName}</td>
              </tr>
              <tr>
                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                  ブランド
                </th>
                <td className="py-2">{brand}</td>
              </tr>
              <tr>
                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                  シンボル
                </th>
                <td className="py-2">{symbol}</td>
              </tr>
              <tr>
                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                  申請者
                </th>
                <td className="py-2">{requestedBy}</td>
              </tr>
              <tr>
                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                  申請日時
                </th>
                <td className="py-2">{requestDate}</td>
              </tr>
              <tr>
                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                  申請数量
                </th>
                <td className="py-2">{quantity.toLocaleString()}</td>
              </tr>
              <tr>
                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                  ステータス
                </th>
                <td className="py-2">
                  {status === "承認待ち" && (
                    <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-semibold bg-yellow-100 text-yellow-800">
                      承認待ち
                    </span>
                  )}
                  {status === "承認済み" && (
                    <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-semibold bg-green-100 text-green-800">
                      承認済み
                    </span>
                  )}
                  {status === "却下" && (
                    <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-semibold bg-red-100 text-red-800">
                      却下
                    </span>
                  )}
                </td>
              </tr>
              <tr>
                <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                  備考
                </th>
                <td className="py-2">{remarks}</td>
              </tr>
            </tbody>
          </table>
        </CardContent>
      </Card>
    </div>
  );
}
