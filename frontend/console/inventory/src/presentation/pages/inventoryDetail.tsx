// frontend/console/inventory/src/presentation/pages/inventoryDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";
import ProductBlueprintCard from "../../../../productBlueprint/src/presentation/components/productBlueprintCard";
import InventoryCard from "../components/inventoryCard";

import { useInventoryDetail } from "../hook/useInventoryDetail";

export default function InventoryDetail() {
  const navigate = useNavigate();

  // ✅ 方針A: URL で pbId + tbId を受け取る
  const { productBlueprintId, tokenBlueprintId } = useParams<{
    productBlueprintId?: string;
    tokenBlueprintId?: string;
  }>();

  /**
   * ★ pbId/tbId が無い（＝ /inventory/detail だけ or 旧ルートに誤マッチ）
   *    → 一覧ページへ強制リダイレクト
   */
  React.useEffect(() => {
    if (!productBlueprintId || !tokenBlueprintId) {
      console.warn(
        "[inventory/InventoryDetail] missing productBlueprintId or tokenBlueprintId -> redirect",
        { productBlueprintId, tokenBlueprintId },
      );
      navigate("/inventory", { replace: true });
    }
  }, [productBlueprintId, tokenBlueprintId, navigate]);

  // ★ 戻るボタンも常に一覧へ戻す
  const onBack = React.useCallback(() => {
    navigate("/inventory");
  }, [navigate]);

  // ✅ hook（方針A）: pbId + tbId -> inventoryIds -> details -> merge
  const { rows, loading, error, vm } = useInventoryDetail(
    productBlueprintId,
    tokenBlueprintId,
  );

  const title = vm
    ? `在庫詳細：${vm.productBlueprintId} / ${vm.tokenBlueprintId}`
    : `在庫詳細：${productBlueprintId ?? ""} / ${tokenBlueprintId ?? ""}`;

  return (
    <PageStyle
      layout="grid-2"
      title={title}
      onBack={onBack}
      onSave={undefined}
    >
      {/* 左カラム：商品情報カード + 在庫一覧カード */}
      <div>
        <ProductBlueprintCard mode="view" />

        {/* --- style elements only --- */}
        {loading && (
          <div className="text-sm text-[hsl(var(--muted-foreground))] mt-2">
            読み込み中...
          </div>
        )}

        {error && (
          <div className="text-sm text-red-600 mt-2">
            読み込みに失敗しました: {error}
          </div>
        )}
        {/* --- style elements only --- */}

        <InventoryCard rows={rows} />
      </div>

      {/* 右カラム：AdminCard のみ */}
      <AdminCard title="管理情報" />
    </PageStyle>
  );
}
