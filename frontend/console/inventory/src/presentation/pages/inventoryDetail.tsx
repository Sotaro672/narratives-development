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
      navigate("/inventory", { replace: true });
    }
  }, [productBlueprintId, tokenBlueprintId, navigate]);

  // ★ 戻るボタンは常に一覧へ戻す
  const onBack = React.useCallback(() => {
    navigate("/inventory");
  }, [navigate]);

  // ✅ hook（方針A）: pbId + tbId -> inventoryIds -> details -> merge
  const { rows, loading, error, vm, tokenBlueprintPatch } = useInventoryDetail(
    productBlueprintId,
    tokenBlueprintId,
  );

  const pbId = s(vm?.productBlueprintId ?? productBlueprintId);
  const tbId = s(vm?.tokenBlueprintId ?? tokenBlueprintId);
  const pbPatch = vm?.productBlueprintPatch;

  const title = pbPatch?.productName
    ? `在庫詳細：${pbPatch.productName}`
    : vm
      ? `在庫詳細：${vm.productBlueprintId} / ${vm.tokenBlueprintId}`
      : `在庫詳細：${productBlueprintId ?? ""} / ${tokenBlueprintId ?? ""}`;

  // ✅ 出品ボタン: /inventory/list/create/:productBlueprintId/:tokenBlueprintId へ
  const onList = React.useCallback(() => {
    if (!pbId || !tbId) return;
    navigate(`/inventory/list/create/${pbId}/${tbId}`);
  }, [navigate, pbId, tbId]);

  // ============================================================
  // ✅ TokenBlueprintCard (view only)
  // - tokenBlueprintPatch を正とする
  // ============================================================

  const tbPatchAny = React.useMemo(() => {
    return (
      (tokenBlueprintPatch as any) ??
      (vm as any)?.tokenBlueprintPatch ??
      (vm as any)?.tokenBlueprint ??
      (vm as any)?.TokenBlueprint ??
      null
    );
  }, [tokenBlueprintPatch, vm]);

  const tokenCardVM: TokenBlueprintCardViewModel = React.useMemo(() => {
    const tokenName = s(tbPatchAny?.tokenName ?? tbPatchAny?.TokenName);
    const nameFallback = s(tbPatchAny?.name ?? tbPatchAny?.Name);
    const nameToShow = tokenName || nameFallback;

    const symbol = s(tbPatchAny?.symbol ?? tbPatchAny?.Symbol);
    const brandId = s(
      tbPatchAny?.brandId ?? tbPatchAny?.BrandID ?? tbPatchAny?.BrandId,
    );
    const brandName = s(tbPatchAny?.brandName ?? tbPatchAny?.BrandName);
    const description = String(
      tbPatchAny?.description ?? tbPatchAny?.Description ?? "",
    );

    const iconUrl =
      s(tbPatchAny?.iconUrl) ||
      s(tbPatchAny?.iconURL) ||
      s(tbPatchAny?.IconURL) ||
      undefined;

    // minted は在庫詳細では view-only に寄せる（trueにすると編集UIが出るため）
    const minted = false;

    return {
      id: tbId,
      name: nameToShow,
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
  }, [tbId, tbPatchAny]);

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
