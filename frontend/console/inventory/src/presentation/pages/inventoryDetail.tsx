// frontend/inventory/src/presentation/pages/inventoryDetail.tsx
import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";
import ProductBlueprintCard from "../../../../productBlueprint/src/presentation/components/productBlueprintCard";

export default function InventoryDetail() {
  const navigate = useNavigate();
  const { inventoryId } = useParams<{ inventoryId?: string }>();

  /**
   * ★ inventoryId が無い（＝ /inventory/detail だけ or /inventory に誤マッチ）
   *    → 一覧ページへ強制リダイレクト
   */
  React.useEffect(() => {
    if (!inventoryId) {
      navigate("/inventory", { replace: true });
    }
  }, [inventoryId, navigate]);

  // ★ 戻るボタンも常に一覧へ戻す
  const onBack = React.useCallback(() => {
    navigate("/inventory");
  }, [navigate]);

  return (
    <PageStyle
      layout="grid-2"
      title={`在庫詳細：${inventoryId ?? ""}`}
      onBack={onBack}
      onSave={undefined}
    >
      {/* 左カラム：商品情報カードのみ */}
      <div>
        <ProductBlueprintCard mode="view" />
      </div>

      {/* 右カラム：AdminCard のみ */}
      <AdminCard title="管理情報" />
    </PageStyle>
  );
}
