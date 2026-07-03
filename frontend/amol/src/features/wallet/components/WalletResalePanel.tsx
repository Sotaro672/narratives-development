// frontend/amol/src/features/wallet/components/WalletResalePanel.tsx
import { useCallback, useEffect, useMemo, useState } from "react";

import {
  listMyResaleConditionImages,
  listMyResaleListings,
} from "../../resale/api/resaleApi";
import type {
  ResaleConditionImage,
  ResaleListing,
} from "../../resale/api/resaleApi";

type ResaleImageMap = Record<string, string>;

function formatPrice(value: number | undefined): string {
  const price = Number(value ?? 0);

  if (!Number.isFinite(price) || price <= 0) {
    return "-";
  }

  return `¥${price.toLocaleString("ja-JP")}`;
}

function formatDateTime(value: string | undefined | null): string {
  if (!value) {
    return "-";
  }

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return "-";
  }

  return new Intl.DateTimeFormat("ja-JP", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

function textOrEmpty(value: string | undefined | null): string {
  return String(value ?? "").trim();
}

function getPrimaryImageUrl(
  item: ResaleListing,
  images: ResaleConditionImage[],
): string {
  if (images.length === 0) {
    return "";
  }

  const primaryImageId = String(item.imageId ?? "").trim();

  if (primaryImageId) {
    const primary = images.find((image) => image.id === primaryImageId);

    if (primary?.url) {
      return primary.url;
    }
  }

  const sortedImages = [...images].sort((a, b) => {
    const aOrder = Number(a.displayOrder ?? 0);
    const bOrder = Number(b.displayOrder ?? 0);

    if (aOrder !== bOrder) {
      return aOrder - bOrder;
    }

    return String(a.id || "").localeCompare(String(b.id || ""), "ja");
  });

  return sortedImages[0]?.url || "";
}

export default function WalletResalePanel() {
  const [items, setItems] = useState<ResaleListing[]>([]);
  const [imageUrlByResaleId, setImageUrlByResaleId] = useState<ResaleImageMap>(
    {},
  );
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string>("");

  const hasItems = items.length > 0;

  const sortedItems = useMemo(() => {
    return [...items].sort((a, b) => {
      const aTime = new Date(a.updatedAt || a.createdAt || "").getTime();
      const bTime = new Date(b.updatedAt || b.createdAt || "").getTime();

      if (Number.isNaN(aTime) && Number.isNaN(bTime)) {
        return String(b.id || "").localeCompare(String(a.id || ""), "ja");
      }

      if (Number.isNaN(aTime)) {
        return 1;
      }

      if (Number.isNaN(bTime)) {
        return -1;
      }

      return bTime - aTime;
    });
  }, [items]);

  const loadResaleImages = useCallback(
    async (nextItems: ResaleListing[]): Promise<ResaleImageMap> => {
      const entries = await Promise.all(
        nextItems.map(async (item) => {
          const resaleId = String(item.id ?? "").trim();

          if (!resaleId) {
            return null;
          }

          try {
            const images = await listMyResaleConditionImages(resaleId);
            const imageUrl = getPrimaryImageUrl(item, images);

            return [resaleId, imageUrl] as const;
          } catch {
            return [resaleId, ""] as const;
          }
        }),
      );

      const nextMap: ResaleImageMap = {};

      for (const entry of entries) {
        if (!entry) {
          continue;
        }

        const [resaleId, imageUrl] = entry;
        nextMap[resaleId] = imageUrl;
      }

      return nextMap;
    },
    [],
  );

  const loadResales = useCallback(async () => {
    setLoading(true);
    setError("");

    try {
      const result = await listMyResaleListings({
        page: 1,
        perPage: 50,
      });

      const nextItems = result.items ?? [];
      const nextImageMap = await loadResaleImages(nextItems);

      setItems(nextItems);
      setImageUrlByResaleId(nextImageMap);
    } catch (error) {
      setError(
        error instanceof Error
          ? error.message
          : "出品一覧の取得に失敗しました。",
      );
    } finally {
      setLoading(false);
    }
  }, [loadResaleImages]);

  useEffect(() => {
    void loadResales();
  }, [loadResales]);

  if (loading) {
    return (
      <div className="wallet-resale-list">
        <p className="wallet-page__message">読み込み中です...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="wallet-resale-list">
        <div role="alert" className="wallet-page__message">
          <p>{error}</p>
          <button
            type="button"
            className="page-button page-button--secondary"
            onClick={() => void loadResales()}
          >
            再読み込み
          </button>
        </div>
      </div>
    );
  }

  if (!hasItems) {
    return (
      <div className="wallet-resale-list">
        <div className="wallet-page__message">
          <p>出品中の商品はありません。</p>
        </div>
      </div>
    );
  }

  return (
    <div className="wallet-resale-list">
      {sortedItems.map((item) => {
        const resaleId = String(item.id ?? "").trim();
        const imageUrl = resaleId ? imageUrlByResaleId[resaleId] || "" : "";

        const productName = textOrEmpty(item.productName);
        const tokenName = textOrEmpty(item.tokenName);
        const brandName = textOrEmpty(item.brandName);

        return (
          <article
            key={resaleId || item.mintAddress}
            className="wallet-resale-list__item"
          >
            <div className="wallet-resale-card">
              <div className="wallet-resale-card__media">
                {imageUrl ? (
                  <img
                    src={imageUrl}
                    alt={productName || tokenName || brandName || "出品画像"}
                    className="wallet-resale-card__image"
                    loading="lazy"
                  />
                ) : (
                  <div
                    className="wallet-resale-card__image-placeholder"
                    aria-label="画像未設定"
                  >
                    画像未設定
                  </div>
                )}
              </div>

              <div className="wallet-resale-card__body">
                <div className="wallet-resale-card__summary">
                  {productName ? (
                    <p className="wallet-resale-card__product-name">
                      {productName}
                    </p>
                  ) : null}

                  {tokenName ? (
                    <p className="wallet-resale-card__token-name">
                      {tokenName}
                    </p>
                  ) : null}

                  {brandName ? (
                    <p className="wallet-resale-card__brand-name">
                      {brandName}
                    </p>
                  ) : null}
                </div>

                <div className="wallet-resale-card__values">
                  <p className="wallet-resale-card__price">
                    {formatPrice(item.price)}
                  </p>

                  <p className="wallet-resale-card__date">
                    {formatDateTime(item.createdAt)}
                  </p>
                </div>
              </div>
            </div>
          </article>
        );
      })}
    </div>
  );
}