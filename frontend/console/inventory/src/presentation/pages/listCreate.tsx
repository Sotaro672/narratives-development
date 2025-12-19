// frontend/console/inventory/src/presentation/pages/listCreate.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";

function s(v: unknown): string {
  return String(v ?? "").trim();
}

export default function InventoryListCreate() {
  const navigate = useNavigate();

  // ✅ routes.tsx で定義した param を受け取る（inventoryId or pb/tb）
  const params = useParams<{
    inventoryId?: string;
    productBlueprintId?: string;
    tokenBlueprintId?: string;
  }>();

  const inventoryId = s(params.inventoryId);
  const productBlueprintId = s(params.productBlueprintId);
  const tokenBlueprintId = s(params.tokenBlueprintId);

  const title = inventoryId
    ? `出品作成（inventoryId: ${inventoryId}）`
    : productBlueprintId && tokenBlueprintId
      ? `出品作成（pb: ${productBlueprintId} / tb: ${tokenBlueprintId}）`
      : "出品作成";

  const onBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  return (
    <PageStyle layout="grid-2" title={title} onBack={onBack}>
      {/* 左カラム：空（grid-2 のレイアウト維持） */}
      <div />

      {/* 右カラム：空（grid-2 のレイアウト維持） */}
      <div />
    </PageStyle>
  );
}
