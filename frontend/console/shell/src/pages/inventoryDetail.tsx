// frontend/console/shell/src/pages/inventoryDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

import PageStyle from "../layout/PageStyle/PageStyle";

import ProductBlueprintCard, {
  type ProductBlueprintPatchInput,
} from "../features/productBlueprint/presentation/cards/productBlueprintForm";

import InventoryCard from "../features/inventory/presentation/components/inventoryCard";
import InventoryShippingAddressCard from "../features/inventory/presentation/components/InventoryShippingAddressCard";
import InventoryListCard from "../features/inventory/presentation/components/InventoryListCard";

import TokenBlueprintCard, {
  type TokenBlueprintCardViewModel,
} from "../features/tokenBlueprint/presentation/components/tokenBlueprintCard";

import { useInventoryDetail } from "../features/inventory/presentation/hook/useInventoryDetail";

export default function InventoryDetail() {
  const navigate = useNavigate();

  /**
   * URLではinventoryIdのみを受け取る。
   */
  const { inventoryId: inventoryIdParam } = useParams<{
    inventoryId?: string;
  }>();

  const inventoryId = inventoryIdParam ?? "";

  /**
   * inventoryIdが存在しない場合は、
   * 在庫一覧画面へ戻す。
   */
  React.useEffect(() => {
    if (!inventoryId) {
      navigate("/inventory", { replace: true });
    }
  }, [inventoryId, navigate]);

  /**
   * 戻るボタンでは在庫一覧画面へ遷移する。
   */
  const onBack = React.useCallback(() => {
    navigate("/inventory");
  }, [navigate]);

  const {
    rows,
    loading,
    error,
    vm,
    selectedShippingAddressId,
    shippingAddressOptions,
    shippingAddressSaving,
    shippingAddressError,
    listItems,
    listLoading,
    listError,
    handleSelectShippingAddress,
    handleSaveShippingAddress,
  } = useInventoryDetail(inventoryId);

  const title = vm?.headerTitle ? `在庫詳細：${vm.headerTitle}` : "在庫詳細";

  /**
   * 出品作成画面へ遷移する。
   */
  const onList = React.useCallback(() => {
    if (!inventoryId) {
      return;
    }

    navigate(`/inventory/list/create/${encodeURIComponent(inventoryId)}`);
  }, [inventoryId, navigate]);

  /**
   * 在庫保管場所は選択しただけでは決定済みとしない。
   * Backendへ保存済みのshippingAddressIdと現在選択中のIDが一致している場合のみ、
   * 出品可能な状態として扱う。
   */
  const hasConfirmedShippingAddress =
    Boolean(vm?.shippingAddressId) &&
    vm?.shippingAddressId === selectedShippingAddressId;

  /**
   * ProductBlueprintPatchDTOは
   * ProductBlueprintPatchInputと互換性があるため、
   * 個別の型変換や型アサーションは行わない。
   */
  const productBlueprintPatchForCard:
    | ProductBlueprintPatchInput
    | undefined = vm?.productBlueprintPatch;

  /**
   * TokenBlueprintCardを参照専用で表示する。
   */
  const tokenBlueprintId = vm?.tokenBlueprintId ?? "";
  const tokenBlueprintPatch = vm?.tokenBlueprintPatch;

  const tokenCardVM = React.useMemo<TokenBlueprintCardViewModel>(() => {
    const tokenName = tokenBlueprintPatch?.tokenName ?? "";

    return {
      id: tokenBlueprintId,
      name: tokenName || tokenBlueprintId || "-",
      symbol: tokenBlueprintPatch?.symbol ?? "",
      brandId: tokenBlueprintPatch?.brandId ?? "",
      brandName: tokenBlueprintPatch?.brandName ?? "",
      description: tokenBlueprintPatch?.description ?? "",
      iconUrl: tokenBlueprintPatch?.iconUrl ?? undefined,
      minted: false,
      iconFile: null,
      isEditMode: false,
      brandOptions: [],
    };
  }, [tokenBlueprintId, tokenBlueprintPatch]);

  return (
    <PageStyle
      layout="grid-2"
      title={title}
      onBack={onBack}
      onSave={handleSaveShippingAddress}
      isSaving={shippingAddressSaving}
    >
      {/* 左カラム */}
      <div>
        <ProductBlueprintCard
          mode="view"
          productBlueprintPatch={productBlueprintPatchForCard}
        />

        {tokenBlueprintId ? (
          <div className="mt-3">
            <TokenBlueprintCard vm={tokenCardVM} />
          </div>
        ) : null}

        {loading ? (
          <div className="mt-2 text-sm text-[hsl(var(--muted-foreground))]">
            読み込み中...
          </div>
        ) : null}

        {error ? (
          <div className="mt-2 text-sm text-red-600">
            読み込みに失敗しました: {error}
          </div>
        ) : null}

        <InventoryCard rows={rows} />
      </div>

      {/* 右カラム */}
      <div className="space-y-4">
        <InventoryShippingAddressCard
          shippingAddressId={selectedShippingAddressId}
          shippingAddressOptions={shippingAddressOptions}
          loading={loading}
          saving={shippingAddressSaving}
          onSelectShippingAddress={handleSelectShippingAddress}
        />

        {shippingAddressError ? (
          <div className="text-sm text-red-600">
            保存に失敗しました: {shippingAddressError}
          </div>
        ) : null}

        {hasConfirmedShippingAddress ? (
          <InventoryListCard
            items={listItems}
            loading={listLoading}
            error={listError}
            onList={onList}
          />
        ) : null}
      </div>
    </PageStyle>
  );
}