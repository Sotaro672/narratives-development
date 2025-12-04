// frontend/console/mintRequest/src/presentation/pages/mintRequestDetail.tsx

import * as React from "react";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import { Card, CardContent } from "../../../../shell/src/shared/ui/card";
import { Button } from "../../../../shell/src/shared/ui/button";
import { Coins } from "lucide-react";

import ProductBlueprintCard from "../../../../productBlueprint/src/presentation/components/productBlueprintCard";
import InspectionResultCard from "../components/inspectionResultCard";
import { useMintRequestDetail } from "../hook/useMintRequestDetail";

import "../styles/mintRequest.css";

export default function MintRequestDetail() {
  const {
    title,
    loading,
    error,
    inspectionCardData,
    blueprint, // 将来 TokenBlueprintCard で使う予定だが、現状は表示しない
    totalMintQuantity,
    onBack,
    handleMint,
  } = useMintRequestDetail();

  return (
    <PageStyle layout="grid-2" title={title} onBack={onBack}>
      {/* 左カラム */}
      <div className="space-y-4 mt-4">
        {/* ① プロダクト基本情報（閲覧モード）
            ※ まだ inspectionCardData に productName などの項目を実装していないため、
               現時点では空の閲覧用カードとして配置しておく */}
        <ProductBlueprintCard mode="view" />

        {/* ② 検査結果カード */}
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

        {/* ③ （将来用）TokenBlueprintCard はデフォルト非表示 */}
        {false && blueprint && (
          <div className="mt-4">
            {/* TokenBlueprintCard を実装する際にここで blueprint を利用する */}
          </div>
        )}

        {/* ④ ミント申請ボタン */}
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
