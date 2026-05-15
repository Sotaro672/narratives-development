// frontend/console/inventory/src/presentation/pages/inventoryDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import ProductBlueprintCard from "../../../../productBlueprint/src/presentation/cards/productBlueprintForm";
import InventoryCard from "../components/inventoryCard";

// TokenBlueprintCard（view-only）
import TokenBlueprintCard, {
  type TokenBlueprintCardViewModel,
  type TokenBlueprintCardHandlers,
} from "../../../../tokenBlueprint/src/presentation/components/tokenBlueprintCard";

import { useInventoryDetail } from "../hook/useInventoryDetail";
import type { InventoryDetailViewModel } from "../../application/inventoryDetail/inventoryDetail.types";

type ProductBlueprintCardPatch = React.ComponentProps<
  typeof ProductBlueprintCard
>["productBlueprintPatch"];

type ProductBlueprintCardCategory = NonNullable<
  NonNullable<ProductBlueprintCardPatch>["productBlueprintCategory"]
>;

function toProductBlueprintCardPatch(
  patch: InventoryDetailViewModel["productBlueprintPatch"] | undefined,
): ProductBlueprintCardPatch {
  if (!patch) return undefined;

  const category = patch.productBlueprintCategory;

  return {
    ...patch,
    productBlueprintCategory: category
      ? ({
          ...category,
          kind: category.kind,
        } as ProductBlueprintCardCategory)
      : category,
  } as ProductBlueprintCardPatch;
}

export default function InventoryDetail() {
  const navigate = useNavigate();

  // 新方針: URL は inventoryId(docId) のみ
  const { inventoryId: inventoryIdParam } = useParams<{
    inventoryId?: string;
  }>();
  const inventoryId = inventoryIdParam ?? "";

  /**
   * inventoryId が無い（= /inventory/detail だけ or 旧ルートに誤マッチ）
   * → 一覧ページへ強制リダイレクト
   */
  React.useEffect(() => {
    if (!inventoryId) {
      navigate("/inventory", { replace: true });
    }
  }, [inventoryId, navigate]);

  // 戻るボタンは常に一覧へ戻す
  const onBack = React.useCallback(() => {
    navigate("/inventory");
  }, [navigate]);

  // hook（inventoryId 前提）
  const { rows, loading, error, vm } = useInventoryDetail(inventoryId);

  // Header は productName/tokenName のみ
  const title = vm?.headerTitle ? `在庫詳細：${vm.headerTitle}` : "在庫詳細";

  // 出品ボタン: /inventory/list/create/:inventoryId
  const onList = React.useCallback(() => {
    if (!inventoryId) return;
    navigate(`/inventory/list/create/${encodeURIComponent(inventoryId)}`);
  }, [navigate, inventoryId]);

  const productBlueprintPatchForCard = React.useMemo(
    () => toProductBlueprintCardPatch(vm?.productBlueprintPatch),
    [vm?.productBlueprintPatch],
  );

  // ============================================================
  // TokenBlueprintCard (view only)
  // - TokenBlueprintCardViewModel の minted は必須なので必ず渡す
  // - この画面では編集しないので minted=false に固定
  // ============================================================

  const tbId = vm?.tokenBlueprintId ?? "";
  const tbPatch = vm?.tokenBlueprintPatch;

  const tokenCardVM: TokenBlueprintCardViewModel = React.useMemo(() => {
    const tokenName = tbPatch?.tokenName ?? "";
    const symbol = tbPatch?.symbol ?? "";
    const brandName = tbPatch?.brandName ?? "";
    const description = tbPatch?.description ?? "";
    const iconUrl = tbPatch?.iconUrl ?? undefined;

    // TokenBlueprintCard 側が brandId 必須なら空文字で埋める
    const brandId = "";

    // minted は必須。詳細画面では編集UI不要なので false 固定。
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
          productBlueprintPatch={productBlueprintPatchForCard}
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