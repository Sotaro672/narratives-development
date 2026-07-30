// frontend/console/shell/src/pages/inventoryDetail.tsx

import * as React from "react";
import {
  useNavigate,
  useParams,
} from "react-router-dom";

import PageStyle from "../layout/PageStyle/PageStyle";

import ProductBlueprintCard from "../features/productBlueprint/presentation/cards/productBlueprintForm";
import InventoryCard from "../features/inventory/presentation/components/inventoryCard";

import TokenBlueprintCard, {
  type TokenBlueprintCardHandlers,
  type TokenBlueprintCardViewModel,
} from "../features/tokenBlueprint/presentation/components/tokenBlueprintCard";

import {
  useInventoryDetail,
} from "../features/inventory/presentation/hook/useInventoryDetail";

import type {
  InventoryDetailViewModel,
} from "../features/inventory/application/inventoryDetail/inventoryDetail.types";

type ProductBlueprintCardPatch =
  React.ComponentProps<
    typeof ProductBlueprintCard
  >["productBlueprintPatch"];

type ProductBlueprintCardCategory =
  NonNullable<
    NonNullable<
      ProductBlueprintCardPatch
    >["productBlueprintCategory"]
  >;

/**
 * InventoryDetailではTokenBlueprintCardを
 * 参照専用として使用する。
 *
 * TokenBlueprintCardのhandlersは必須だが、
 * この画面では編集処理を行わないため空オブジェクトを渡す。
 */
const READ_ONLY_TOKEN_BLUEPRINT_CARD_HANDLERS:
  TokenBlueprintCardHandlers = {};

function toProductBlueprintCardPatch(
  patch:
    | InventoryDetailViewModel["productBlueprintPatch"]
    | undefined,
): ProductBlueprintCardPatch {
  if (!patch) {
    return undefined;
  }

  const category =
    patch.productBlueprintCategory;

  return {
    ...patch,

    productBlueprintCategory:
      category
        ? ({
            ...category,
            kind: category.kind,
          } as ProductBlueprintCardCategory)
        : category,
  } as ProductBlueprintCardPatch;
}

export default function InventoryDetail() {
  const navigate =
    useNavigate();

  /**
   * URLではinventoryIdのみを受け取る。
   */
  const {
    inventoryId:
      inventoryIdParam,
  } = useParams<{
    inventoryId?: string;
  }>();

  const inventoryId =
    inventoryIdParam ?? "";

  /**
   * inventoryIdが存在しない場合は、
   * 在庫一覧画面へ戻す。
   */
  React.useEffect(() => {
    if (!inventoryId) {
      navigate(
        "/inventory",
        {
          replace: true,
        },
      );
    }
  }, [
    inventoryId,
    navigate,
  ]);

  /**
   * 戻るボタンでは在庫一覧画面へ遷移する。
   */
  const onBack =
    React.useCallback(() => {
      navigate(
        "/inventory",
      );
    }, [navigate]);

  const {
    rows,
    loading,
    error,
    vm,
  } = useInventoryDetail(
    inventoryId,
  );

  const title =
    vm?.headerTitle
      ? `在庫詳細：${vm.headerTitle}`
      : "在庫詳細";

  /**
   * 出品作成画面へ遷移する。
   */
  const onList =
    React.useCallback(() => {
      if (!inventoryId) {
        return;
      }

      navigate(
        `/inventory/list/create/${encodeURIComponent(
          inventoryId,
        )}`,
      );
    }, [
      navigate,
      inventoryId,
    ]);

  const productBlueprintPatchForCard =
    React.useMemo(
      () =>
        toProductBlueprintCardPatch(
          vm?.productBlueprintPatch,
        ),
      [
        vm?.productBlueprintPatch,
      ],
    );

  /**
   * TokenBlueprintCardを参照専用で表示する。
   */
  const tokenBlueprintId =
    vm?.tokenBlueprintId ?? "";

  const tokenBlueprintPatch =
    vm?.tokenBlueprintPatch;

  const tokenCardVM:
    TokenBlueprintCardViewModel =
    React.useMemo(() => {
      const tokenName =
        tokenBlueprintPatch
          ?.tokenName ?? "";

      const symbol =
        tokenBlueprintPatch
          ?.symbol ?? "";

      const brandName =
        tokenBlueprintPatch
          ?.brandName ?? "";

      const description =
        tokenBlueprintPatch
          ?.description ?? "";

      const iconUrl =
        tokenBlueprintPatch
          ?.iconUrl ??
        undefined;

      return {
        id:
          tokenBlueprintId,

        name:
          tokenName ||
          tokenBlueprintId ||
          "-",

        symbol,

        /**
         * InventoryDetailViewModelではbrandIdを
         * 保持していないため空文字を設定する。
         */
        brandId:
          "",

        brandName,
        description,
        iconUrl,

        /**
         * この画面では編集しないため、
         * Mint済み状態による編集制御は使用しない。
         */
        minted:
          false,

        iconFile:
          null,

        isEditMode:
          false,

        brandOptions:
          [],
      };
    }, [
      tokenBlueprintId,
      tokenBlueprintPatch,
    ]);

  return (
    <PageStyle
      layout="grid-2"
      title={title}
      onBack={onBack}
      onSave={undefined}
      onList={onList}
    >
      {/* 左カラム */}
      <div>
        <ProductBlueprintCard
          mode="view"
          productBlueprintPatch={
            productBlueprintPatchForCard
          }
        />

        {tokenBlueprintId && (
          <div className="mt-3">
            <TokenBlueprintCard
              vm={tokenCardVM}
              handlers={
                READ_ONLY_TOKEN_BLUEPRINT_CARD_HANDLERS
              }
            />
          </div>
        )}

        {loading && (
          <div className="text-sm text-[hsl(var(--muted-foreground))] mt-2">
            読み込み中...
          </div>
        )}

        {error && (
          <div className="text-sm text-red-600 mt-2">
            読み込みに失敗しました:{" "}
            {error}
          </div>
        )}

        <InventoryCard
          rows={rows}
        />
      </div>

      {/* 右カラム：grid-2維持用 */}
      <div />
    </PageStyle>
  );
}