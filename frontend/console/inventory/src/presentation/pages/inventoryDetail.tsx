// frontend/console/inventory/src/presentation/pages/inventoryDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import ProductBlueprintCard from "../../../../productBlueprint/src/presentation/components/productBlueprintCard";
import InventoryCard from "../components/inventoryCard";

// ✅ TokenBlueprintCard（view-only）
import TokenBlueprintCard, {
  type TokenBlueprintCardViewModel,
  type TokenBlueprintCardHandlers,
} from "../../../../tokenBlueprint/src/presentation/components/tokenBlueprintCard";

import { useInventoryDetail } from "../hook/useInventoryDetail";

function s(v: unknown): string {
  return String(v ?? "").trim();
}

export default function InventoryDetail() {
  const navigate = useNavigate();

  // ✅ 新方針: URL は inventoryId(docId) のみ
  const { inventoryId: inventoryIdParam } = useParams<{ inventoryId?: string }>();
  const inventoryId = s(inventoryIdParam);

  /**
   * ★ inventoryId が無い（＝ /inventory/detail だけ or 旧ルートに誤マッチ）
   *    → 一覧ページへ強制リダイレクト
   */
  React.useEffect(() => {
    if (!inventoryId) {
      navigate("/inventory", { replace: true });
    }
  }, [inventoryId, navigate]);

  // ★ 戻るボタンは常に一覧へ戻す
  const onBack = React.useCallback(() => {
    navigate("/inventory");
  }, [navigate]);

  // ✅ hook（inventoryId 前提）
  const { rows, loading, error, vm } = useInventoryDetail(inventoryId);

  // ✅ Header は productName/tokenName のみ
  const title = s(vm?.headerTitle) ? `在庫詳細：${vm!.headerTitle}` : "在庫詳細";

  // ✅ 出品ボタン: /inventory/list/create/:inventoryId
  const onList = React.useCallback(() => {
    if (!inventoryId) return;
    navigate(`/inventory/list/create/${encodeURIComponent(inventoryId)}`);
  }, [navigate, inventoryId]);

  // ============================================================
  // ✅ TokenBlueprintCard (view only)
  // - TokenBlueprintCardViewModel の minted は必須なので必ず渡す
  // - この画面では編集しないので minted=false に固定（= view-onlyで安全）
  // ============================================================

  const tbId = s(vm?.tokenBlueprintId);
  const tbPatch = vm?.tokenBlueprintPatch;

  const tokenCardVM: TokenBlueprintCardViewModel = React.useMemo(() => {
    const tokenName = s((tbPatch as any)?.tokenName);
    const symbol = s((tbPatch as any)?.symbol);
    const brandName = s((tbPatch as any)?.brandName);
    const description = String((tbPatch as any)?.description ?? "");
    const iconUrl = s((tbPatch as any)?.iconUrl) || undefined;

    // ✅ TokenBlueprintCard 側が brandId 必須なら空文字で埋める
    const brandId = "";

    // ✅ minted は必須。詳細画面では編集UI不要なので false 固定。
    const minted = false;

    return {
      id: tbId,
      name: tokenName || tbId || "-",
      symbol,
      brandId,
      brandName,
      description,
      iconUrl,

      minted,
      iconFile: null,
      isEditMode: false,
      brandOptions: [],
    };
  }, [tbId, tbPatch]);

  const tokenCardHandlers: TokenBlueprintCardHandlers = React.useMemo(
    () => ({
      onPreview: () => {
        const url = tokenCardVM.iconUrl;
        if (url) window.open(url, "_blank", "noopener,noreferrer");
      },
    }),
    [tokenCardVM.iconUrl],
  );

  return (
    <PageStyle
      layout="grid-2"
      title={title}
      onBack={onBack}
      onSave={undefined}
      onList={onList}
    >
      {/* 左カラム：商品情報カード + TokenBlueprintCard + 在庫一覧カード */}
      <div>
        <ProductBlueprintCard
          mode="view"
          productBlueprintPatch={vm?.productBlueprintPatch}
        />

        {/* TokenBlueprintCard */}
        {tbId && (
          <div className="mt-3">
            <TokenBlueprintCard vm={tokenCardVM} handlers={tokenCardHandlers} />
          </div>
        )}

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

        <InventoryCard rows={rows} />
      </div>

      {/* 右カラム：空要素（grid-2維持） */}
      <div />
    </PageStyle>
  );
}
