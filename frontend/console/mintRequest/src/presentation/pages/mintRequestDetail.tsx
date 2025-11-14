// frontend/mintRequest/src/presentation/pages/mintRequestDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";
import InventoryCard, {
  type InventoryRow,
} from "../../../../inventory/src/presentation/components/inventoryCard";
import TokenBlueprintCard from "../../../../tokenBlueprint/src/presentation/components/tokenBlueprintCard";
import TokenContentsCard from "../../../../tokenContents/src/presentation/components/tokenContentsCard";
import { TOKEN_BLUEPRINTS } from "../../../../tokenBlueprint/src/infrastructure/mockdata/tokenBlueprint_mockdata";
import type { TokenBlueprint } from "../../../../tokenBlueprint/src/domain/entity/tokenBlueprint";
import { Card, CardContent } from "../../../../shell/src/shared/ui/card";
import { Button } from "../../../../shell/src/shared/ui/button";
import { Coins } from "lucide-react";

import "../styles/mintRequest.css";

export default function MintRequestDetail() {
  const navigate = useNavigate();
  const { requestId } = useParams<{ requestId: string }>();

  // 管理情報（モック）
  const [assignee, setAssignee] = React.useState("member_sato");
  const [creator] = React.useState("member_yamada");
  const [createdAt] = React.useState("2025-11-05T00:00:00Z");

  // 在庫データ（モデル別在庫一覧：モック）
  const [inventoryRows] = React.useState<InventoryRow[]>([
    {
      modelCode: "LM-SB-S-WHT",
      size: "S",
      colorName: "ホワイト",
      colorCode: "#ffffff",
      stock: 25,
    },
    {
      modelCode: "LM-SB-M-WHT",
      size: "M",
      colorName: "ホワイト",
      colorCode: "#ffffff",
      stock: 42,
    },
    {
      modelCode: "LM-SB-L-WHT",
      size: "L",
      colorName: "ホワイト",
      colorCode: "#ffffff",
      stock: 38,
    },
    {
      modelCode: "LM-SB-M-BLK",
      size: "M",
      colorName: "ブラック",
      colorCode: "#000000",
      stock: 35,
    },
    {
      modelCode: "LM-SB-L-BLK",
      size: "L",
      colorName: "ブラック",
      colorCode: "#000000",
      stock: 28,
    },
    {
      modelCode: "LM-SB-M-NVY",
      size: "M",
      colorName: "ネイビー",
      colorCode: "#1e3a8a",
      stock: 31,
    },
    {
      modelCode: "LM-SB-L-NVY",
      size: "L",
      colorName: "ネイビー",
      colorCode: "#1e3a8a",
      stock: 22,
    },
  ]);

  // 在庫数合計（ミント数）
  const totalStock = React.useMemo(
    () => inventoryRows.reduce((sum, r) => sum + (r.stock || 0), 0),
    [inventoryRows],
  );

  // トークン設計（暫定: 先頭 / 本来は requestId に紐付け）
  const blueprint: TokenBlueprint | undefined = TOKEN_BLUEPRINTS[0];

  // 戻るボタン
  const onBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  // ミント申請ボタン
  const handleMint = React.useCallback(() => {
    alert(
      `ミント申請を実行しました（申請ID: ${
        requestId ?? "不明"
      } / ミント数: ${totalStock}）`,
    );
  }, [requestId, totalStock]);

  return (
    <PageStyle
      layout="grid-2"
      title={`ミント申請詳細：${requestId ?? "不明ID"}`}
      onBack={onBack}
    >
      {/* 左カラム：在庫一覧 → TokenBlueprintCard → TokenContentsCard → ミント申請ボタン */}
      <div className="space-y-4 mt-4">
        <InventoryCard rows={inventoryRows} />

        {blueprint && (
          <TokenBlueprintCard
            initialEditMode={false}
            initialTokenBlueprint={blueprint}
            // ドメイン外拡張フィールド（任意・モック）
            initialBurnAt=""
            initialIconUrl={blueprint.iconId ?? ""}
          />
        )}

        {/* TokenContentsCard: TokenBlueprint.contentFiles（ID配列）と連動させる想定 */}
        <TokenContentsCard
          mode="view"
          images={blueprint?.contentFiles ?? []}
        />

        <Card className="mint-request-card">
          <CardContent className="mint-request-card__body">
            <div className="mint-request-card__actions">
              <Button
                onClick={handleMint}
                className="mint-request-card__button flex items-center gap-2"
              >
                <Coins size={16} />
                ミント申請を実行
              </Button>
              <span className="mint-request-card__total">
                ミント数: <strong>{totalStock}</strong>
              </span>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* 右カラム：管理情報カード */}
      <div className="space-y-4 mt-4">
        <AdminCard
          title="管理情報"
          assigneeName={assignee}
          createdByName={creator}
          createdAt={createdAt}
          onEditAssignee={() => setAssignee("member_new")}
          onClickAssignee={() => console.log("assignee clicked:", assignee)}
          onClickCreatedBy={() => console.log("createdBy clicked:", creator)}
        />
      </div>
    </PageStyle>
  );
}
