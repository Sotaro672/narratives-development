// frontend/amol/src/pages/CartPage.tsx
import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import Layout from "../components/layout/Layout";
import {
  fetchCartItemsWithCatalog,
  fetchCurrentAvatarId,
  removeCartItem,
} from "../features/cart/api/cartApi";
import type { CartDisplayItem } from "../features/cart/types";
import {
  calculateCartTotalAmount,
  formatPrice,
  getModelPrice,
  getModelVariation,
  getPrimaryCatalogImage,
} from "../features/cart/utils/cartUtils";
import "../styles/cart-page.css";

const MOBILE_PORTRAIT_MEDIA_QUERY =
  "(max-width: 959px) and (orientation: portrait)";

type CartDisplayItemWithResolvedFields = CartDisplayItem & {
  brandName?: string;
  imageUrl?: string;
  listImage?: string;
  resaleId?: string;
  title?: string;
  productName?: string;
  modelLabel?: string;
  modelNumber?: string;
  modelKind?: string;
  volumeValue?: number;
  volumeUnit?: string;
  colorName?: string;
  size?: string;
  price?: number;
};

function getApiBaseUrl(): string {
  const env = import.meta.env.VITE_API_BASE_URL;

  if (typeof env === "string" && env.trim() !== "") {
    return env.replace(/\/$/, "");
  }

  return "";
}

function normalizeText(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function asResolvedItem(item: CartDisplayItem): CartDisplayItemWithResolvedFields {
  return item as CartDisplayItemWithResolvedFields;
}

function formatAlcoholVolume(item: CartDisplayItem): string {
  const resolvedItem = asResolvedItem(item);

  if (
    typeof resolvedItem.volumeValue === "number" &&
    Number.isFinite(resolvedItem.volumeValue) &&
    resolvedItem.volumeUnit
  ) {
    return `${resolvedItem.volumeValue}${resolvedItem.volumeUnit}`;
  }

  if (resolvedItem.modelLabel) {
    return resolvedItem.modelLabel;
  }

  return "-";
}

function getCartItemBrandName(item: CartDisplayItem): string {
  const resolvedItem = asResolvedItem(item);

  return (
    normalizeText(resolvedItem.catalog?.productBlueprint.brandName) ||
    normalizeText(resolvedItem.brandName) ||
    "ブランド未設定"
  );
}

function getCartItemProductName(item: CartDisplayItem): string {
  const resolvedItem = asResolvedItem(item);

  return (
    normalizeText(resolvedItem.catalog?.productBlueprint.productName) ||
    normalizeText(resolvedItem.productName) ||
    normalizeText(resolvedItem.catalog?.list.title) ||
    normalizeText(resolvedItem.title) ||
    "商品名未設定"
  );
}

function getCartItemListTitle(item: CartDisplayItem): string {
  const resolvedItem = asResolvedItem(item);
  const catalogTitle = normalizeText(resolvedItem.catalog?.list.title);
  const itemTitle = normalizeText(resolvedItem.title);
  const productName = getCartItemProductName(resolvedItem);

  if (catalogTitle && catalogTitle !== productName) {
    return catalogTitle;
  }

  if (itemTitle && itemTitle !== productName) {
    return itemTitle;
  }

  return "";
}

function getCartItemImageUrl(item: CartDisplayItem): string {
  const resolvedItem = asResolvedItem(item);

  return (
    normalizeText(getPrimaryCatalogImage(resolvedItem.catalog)) ||
    normalizeText(resolvedItem.imageUrl) ||
    normalizeText(resolvedItem.listImage)
  );
}

function getCartItemNavigationPath(item: CartDisplayItem): string {
  const resolvedItem = asResolvedItem(item);

  if (resolvedItem.listId) {
    return `/lists/${encodeURIComponent(resolvedItem.listId)}`;
  }

  if (resolvedItem.resaleId) {
    return `/market/resales/${encodeURIComponent(resolvedItem.resaleId)}`;
  }

  return "";
}

export default function CartPage() {
  const navigate = useNavigate();

  const [items, setItems] = useState<CartDisplayItem[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState("");
  const [removingItemKey, setRemovingItemKey] = useState("");
  const [isMobilePortrait, setIsMobilePortrait] = useState(false);

  const apiBaseUrl = useMemo(() => getApiBaseUrl(), []);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    const mobilePortraitQuery = window.matchMedia(MOBILE_PORTRAIT_MEDIA_QUERY);

    const updateMobilePortraitState = () => {
      setIsMobilePortrait(mobilePortraitQuery.matches);
    };

    updateMobilePortraitState();

    if (typeof mobilePortraitQuery.addEventListener === "function") {
      mobilePortraitQuery.addEventListener(
        "change",
        updateMobilePortraitState,
      );

      return () => {
        mobilePortraitQuery.removeEventListener(
          "change",
          updateMobilePortraitState,
        );
      };
    }

    mobilePortraitQuery.addListener(updateMobilePortraitState);

    return () => {
      mobilePortraitQuery.removeListener(updateMobilePortraitState);
    };
  }, []);

  useEffect(() => {
    let cancelled = false;

    async function loadCart() {
      setIsLoading(true);
      setErrorMessage("");

      try {
        const avatarId = await fetchCurrentAvatarId(apiBaseUrl);

        const itemsWithCatalog = await fetchCartItemsWithCatalog({
          apiBaseUrl,
          avatarId,
        });

        if (cancelled) {
          return;
        }

        setItems(itemsWithCatalog);
      } catch (error) {
        if (cancelled) {
          return;
        }

        setItems([]);
        setErrorMessage(
          error instanceof Error
            ? error.message
            : "カートの取得中にエラーが発生しました。",
        );
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    }

    void loadCart();

    return () => {
      cancelled = true;
    };
  }, [apiBaseUrl]);

  const totalAmount = useMemo(() => {
    return calculateCartTotalAmount(items);
  }, [items]);

  const hasItems = items.length > 0;
  const isPurchaseDisabled = !hasItems || isLoading || removingItemKey !== "";

  function handlePurchase() {
    if (isPurchaseDisabled) {
      return;
    }

    navigate("/payments/cart");
  }

  async function handleRemoveItem(item: CartDisplayItem) {
    if (removingItemKey !== "") {
      return;
    }

    setRemovingItemKey(item.itemKey);
    setErrorMessage("");

    try {
      await removeCartItem({
        apiBaseUrl,
        item,
      });

      setItems((currentItems) =>
        currentItems.filter(
          (currentItem) => currentItem.itemKey !== item.itemKey,
        ),
      );
    } catch (error) {
      setErrorMessage(
        error instanceof Error
          ? error.message
          : "カート商品の削除中にエラーが発生しました。",
      );
    } finally {
      setRemovingItemKey("");
    }
  }

  return (
    <Layout
      title="カート"
      titleClickable={false}
      mode="mypage"
      showBackButton
      backTo="/lists"
      showFooter={isMobilePortrait}
      hideHamburgerMenu
      hideSettingsButton
      actionButtonLabel={isMobilePortrait ? undefined : "購入する"}
      onActionButtonClick={isMobilePortrait ? undefined : handlePurchase}
      actionButtonDisabled={isPurchaseDisabled}
      footerProps={
        isMobilePortrait
          ? {
              variant: "action",
              buttonLabel: "購入する",
              disabled: isPurchaseDisabled,
              onButtonClick: handlePurchase,
            }
          : undefined
      }
    >
      <section className="content-page-section cart-page-section-root">
        {isLoading ? (
          <div className="cart-page-empty">
            <div className="cart-page-empty__icon" aria-hidden="true">
              🛒
            </div>

            <h1 className="cart-page-empty__title">カートを読み込んでいます</h1>

            <p className="cart-page-empty__text">
              追加済みのアイテムを確認しています。
            </p>
          </div>
        ) : null}

        {!isLoading && errorMessage ? (
          <div className="cart-page-empty">
            <div className="cart-page-empty__icon" aria-hidden="true">
              ⚠️
            </div>

            <h1 className="cart-page-empty__title">
              カートを取得できませんでした
            </h1>

            <p className="cart-page-empty__text">{errorMessage}</p>
          </div>
        ) : null}

        {!isLoading && !errorMessage && !hasItems ? (
          <div className="cart-page-empty">
            <div className="cart-page-empty__icon" aria-hidden="true">
              🛒
            </div>

            <h1 className="cart-page-empty__title">カートは空です</h1>

            <p className="cart-page-empty__text">
              応援したいリストやアイテムを追加すると、ここに表示されます。
            </p>
          </div>
        ) : null}

        {!isLoading && !errorMessage && hasItems ? (
          <div className="cart-page-content">
            <div className="cart-page-list">
              {items.map((item) => {
                const resolvedItem = asResolvedItem(item);
                const catalog = resolvedItem.catalog;
                const modelId = resolvedItem.modelId || "";
                const model = getModelVariation(catalog, modelId);
                const imageUrl = getCartItemImageUrl(resolvedItem);
                const catalogPrice = getModelPrice(catalog, modelId);
                const price = catalogPrice ?? resolvedItem.price ?? null;
                const lineAmount =
                  price === null ? null : price * resolvedItem.qty;
                const isRemoving = removingItemKey === resolvedItem.itemKey;
                const isAlcohol = resolvedItem.modelKind === "alcohol";
                const brandName = getCartItemBrandName(resolvedItem);
                const productName = getCartItemProductName(resolvedItem);
                const listTitle = getCartItemListTitle(resolvedItem);
                const navigationPath = getCartItemNavigationPath(resolvedItem);
                const canNavigate = navigationPath !== "";

                return (
                  <article
                    key={resolvedItem.itemKey}
                    className="cart-page-item"
                  >
                    <button
                      type="button"
                      className="cart-page-item__remove-button"
                      aria-label="カートから商品を削除"
                      disabled={removingItemKey !== ""}
                      onClick={(event) => {
                        event.stopPropagation();
                        void handleRemoveItem(resolvedItem);
                      }}
                    >
                      {isRemoving ? "…" : "×"}
                    </button>

                    <button
                      type="button"
                      className="cart-page-item__image-button"
                      disabled={!canNavigate}
                      onClick={() => {
                        if (!navigationPath) {
                          return;
                        }

                        navigate(navigationPath);
                      }}
                    >
                      {imageUrl ? (
                        <img
                          src={imageUrl}
                          alt={productName}
                          className="cart-page-item__image"
                        />
                      ) : (
                        <div className="cart-page-item__image-placeholder">
                          No Image
                        </div>
                      )}
                    </button>

                    <div className="cart-page-item__body">
                      <p className="cart-page-item__brand">{brandName}</p>

                      <h2 className="cart-page-item__title">{productName}</h2>

                      {listTitle ? (
                        <p className="cart-page-item__list-title">
                          {listTitle}
                        </p>
                      ) : null}

                      <dl className="cart-page-item__meta">
                        {isAlcohol ? (
                          <>
                            <div>
                              <dt>品番</dt>
                              <dd>{resolvedItem.modelNumber || "-"}</dd>
                            </div>
                            <div>
                              <dt>容量</dt>
                              <dd>{formatAlcoholVolume(resolvedItem)}</dd>
                            </div>
                          </>
                        ) : (
                          <>
                            <div>
                              <dt>カラー</dt>
                              <dd>
                                {model?.colorName ||
                                  resolvedItem.colorName ||
                                  resolvedItem.color ||
                                  "-"}
                              </dd>
                            </div>
                            <div>
                              <dt>サイズ</dt>
                              <dd>{model?.size || resolvedItem.size || "-"}</dd>
                            </div>
                          </>
                        )}

                        <div>
                          <dt>数量</dt>
                          <dd>{resolvedItem.qty}</dd>
                        </div>
                      </dl>

                      <p className="cart-page-item__price">
                        {lineAmount === null
                          ? "価格未設定"
                          : formatPrice(lineAmount)}
                      </p>
                    </div>
                  </article>
                );
              })}
            </div>

            <aside className="cart-page-summary">
              <h2 className="cart-page-summary__title">注文内容</h2>

              <dl className="cart-page-summary__list">
                <div>
                  <dt>商品数</dt>
                  <dd>{items.length}</dd>
                </div>
                <div>
                  <dt>合計</dt>
                  <dd>{formatPrice(totalAmount)}</dd>
                </div>
              </dl>
            </aside>
          </div>
        ) : null}
      </section>
    </Layout>
  );
}