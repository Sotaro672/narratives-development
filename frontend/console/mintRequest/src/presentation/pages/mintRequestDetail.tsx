// frontend/mintRequest/src/presentation/pages/mintRequestDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import TokenBlueprintCard from "../../../../tokenBlueprint/src/presentation/components/tokenBlueprintCard";
import { TOKEN_BLUEPRINTS } from "../../../../tokenBlueprint/src/infrastructure/mockdata/tokenBlueprint_mockdata";
import type { TokenBlueprint } from "../../../../tokenBlueprint/src/domain/entity/tokenBlueprint";
import { Card, CardContent } from "../../../../shell/src/shared/ui/card";
import { Button } from "../../../../shell/src/shared/ui/button";
import { Coins } from "lucide-react";

import InspectionResultCard from "../components/inspectionResultCard";
import { useInspectionResultCard } from "../hook/useInspectionResultCard";
import {
  fetchInspectionByProductionId,
  type InspectionBatchDTO,
} from "../../infrastructure/api/mintRequestApi";

import "../styles/mintRequest.css";

export default function MintRequestDetail() {
  const navigate = useNavigate();
  const { requestId } = useParams<{ requestId: string }>();

  // 検査バッチ（backend: inspections コレクション）
  const [inspectionBatch, setInspectionBatch] =
    React.useState<InspectionBatchDTO | null>(null);
  const [loading, setLoading] = React.useState<boolean>(false);
  const [error, setError] = React.useState<string | null>(null);

  // requestId（= productionId）から InspectionBatch を取得
  React.useEffect(() => {
    if (!requestId) return;

    let cancelled = false;
    const run = async () => {
      setLoading(true);
      setError(null);
      try {
        const batch = await fetchInspectionByProductionId(requestId);
        if (!cancelled) {
          setInspectionBatch(batch);
        }
      } catch (e: any) {
        if (!cancelled) {
          setError(e?.message ?? "検査結果の取得に失敗しました");
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    run();
    return () => {
      cancelled = true;
    };
  }, [requestId]);

  // 検査結果カード用データ（InspectionBatch → モデル別行データ）
  const inspectionCardData = useInspectionResultCard({
    batch: inspectionBatch ?? undefined,
  });

  // ミント数 = 合格数合計（totalPassed）
  const totalMintQuantity = inspectionCardData.totalPassed;

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
      } / ミント数: ${totalMintQuantity}）`,
    );
  }, [requestId, totalMintQuantity]);

  return (
    <PageStyle
      layout="grid-2"
      title={`ミント申請詳細：${requestId ?? "不明ID"}`}
      onBack={onBack}
    >
      {/* 左カラム：検査結果カード → TokenBlueprintCard → ミント申請ボタン */}
      <div className="space-y-4 mt-4">
        {/* モデル別在庫カードの代わりに検査結果カードを表示 */}
        {loading ? (
          <Card className="mint-request-card">
            <CardContent className="mint-request-card__body">
              検査結果を読み込み中です…
            </CardContent>
          </Card>
        ) : error ? (
          <Card className="mint-request-card">
            <CardContent className="mint-request-card__body text-red-600">
              {error}
            </CardContent>
          </Card>
        ) : (
          <InspectionResultCard data={inspectionCardData} />
        )}

        {blueprint && (
          <TokenBlueprintCard
            initialEditMode={false}
            initialTokenBlueprint={blueprint}
            // ドメイン外拡張フィールド（任意・モック）
            initialBurnAt=""
            initialIconUrl={blueprint.iconId ?? ""}
          />
        )}

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
                ミント数: <strong>{totalMintQuantity}</strong>
              </span>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* 右カラム：現状は空（将来、別カードを配置予定） */}
      <div className="space-y-4 mt-4">{/* placeholder */}</div>
    </PageStyle>
  );
}
