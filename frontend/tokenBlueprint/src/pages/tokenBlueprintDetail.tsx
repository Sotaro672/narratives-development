// frontend/tokenBlueprint/src/pages/tokenBlueprintDetail.tsx
import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../admin/src/pages/AdminCard";
import { Card, CardHeader, CardTitle, CardContent } from "../../../shared/ui/card";

export default function TokenBlueprintDetail() {
  const navigate = useNavigate();
  const { tokenBlueprintId } = useParams<{ tokenBlueprintId: string }>();

  // ─────────────────────────────────────────
  // Mock data (表示用)
  // ─────────────────────────────────────────
  const [blueprintName] = React.useState("LUMINA VIP Token");
  const [symbol] = React.useState("LUMI");
  const [network] = React.useState("Solana");
  const [decimals] = React.useState(9);
  const [initialSupply] = React.useState<number>(1_000_000);
  const [mintAuthority] = React.useState<string>("Authority Wallet (xxxx...abcd)");
  const [freezeAuthority] = React.useState<string>("Authority Wallet (xxxx...ef12)");
  const [description] = React.useState(
    "ブランドVIP向けの特典連動トークン。認証/特典配布/購入証明に利用します。"
  );

  // 管理情報（右カラム）
  const [assignee, setAssignee] = React.useState("佐藤 美咲");
  const [creator] = React.useState("佐藤 美咲");
  const [createdAt] = React.useState("2024/05/01");

  // 戻る
  const onBack = React.useCallback(() => navigate(-1), [navigate]);

  return (
    <PageStyle
      layout="grid-2"
      title={`トークン設計：${tokenBlueprintId ?? "不明ID"}`}
      onBack={onBack}
      onSave={undefined} // 必要なら保存ハンドラを差し込んでください
    >
      {/* 左カラム */}
      <div>
        <Card>
          <CardHeader>
            <CardTitle>トークン設計（基本情報）</CardTitle>
          </CardHeader>
          <CardContent>
            <table className="w-full text-sm">
              <tbody>
                <tr>
                  <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                    名称
                  </th>
                  <td className="py-2">{blueprintName}</td>
                </tr>
                <tr>
                  <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                    シンボル
                  </th>
                  <td className="py-2">{symbol}</td>
                </tr>
                <tr>
                  <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                    ネットワーク
                  </th>
                  <td className="py-2">{network}</td>
                </tr>
                <tr>
                  <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                    Decimals
                  </th>
                  <td className="py-2">{decimals}</td>
                </tr>
                <tr>
                  <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                    初期供給量
                  </th>
                  <td className="py-2">{initialSupply.toLocaleString()}</td>
                </tr>
                <tr>
                  <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                    Mint Authority
                  </th>
                  <td className="py-2">{mintAuthority}</td>
                </tr>
                <tr>
                  <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                    Freeze Authority
                  </th>
                  <td className="py-2">{freezeAuthority}</td>
                </tr>
                <tr>
                  <th className="text-muted-foreground font-medium pr-4 py-2 align-top whitespace-nowrap">
                    説明
                  </th>
                  <td className="py-2">{description}</td>
                </tr>
              </tbody>
            </table>
          </CardContent>
        </Card>
      </div>

      {/* 右カラム（管理情報） */}
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
