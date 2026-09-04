// frontend/amol/src/features/wallet/components/WalletResalePanel.tsx

import { useCallback, useEffect, useMemo, useState } from "react";

import { formatDateTime } from "../../../components/utils/date";
import { textOrEmpty } from "../../../components/utils/textOrEmpty";
import {
  listMyResaleConditionImages,
  listMyResaleListings,
  listPublicResaleConditionImages,
  listResaleListingsByAvatarId,
} from "../../resale/api/resaleApi";
import type {
  ResaleConditionImage,
  ResaleListing,
} from "../../resale/api/resaleApi";

type ResaleImageMap = Record<string, string>;

type WalletResalePanelProps = {
  avatarId?: string;
  onItemClick?: (resaleId: string, item: ResaleListing) => void;
};

const STATUS_LABELS: Record<ResaleListing["status"], string> = {
  listing: "出品中",
  suspended: "公開停止",
  sold: "売却済み",
};

const STATUS_CLASS_NAMES: Record<ResaleListing["status"], string> = {
  listing: "wallet-resale-card__media--listing",
  suspended: "wallet-resale-card__media--suspended",
  sold: "wallet-resale-card__media--sold",
};

function formatPrice(value: number): string {
  if (!Number.isFinite(value) || value <= 0) {
    return "-";
  }

  return `¥${value.toLocaleString("ja-JP")}`;
}

function formatStatusLabel(status: ResaleListing["status"]): string {
  return STATUS_LABELS[status];
}

function getStatusClassName(status: ResaleListing["status"]): string {
  return STATUS_CLASS_NAMES[status];
}

function getPrimaryImageUrl(
  item: ResaleListing,
  images: ResaleConditionImage[],
): string {
  if (images.length === 0) {
    return "";
  }

  if (item.imageId) {
    const primary = images.find((image) => image.id === item.imageId);

    if (primary) {
      return primary.url;
    }
  }

  const sortedImages = [...images].sort((a, b) => {
    if (a.displayOrder !== b.displayOrder) {
      return a.displayOrder - b.displayOrder;
    }

    return a.id.localeCompare(b.id, "ja");
  });

  return sortedImages[0]?.url ?? "";
}

export default function WalletResalePanel({
  avatarId,
  onItemClick,
}: WalletResalePanelProps) {
  const normalizedAvatarId = avatarId?.trim() ?? "";
  const isPublicAvatarMode = Boolean(normalizedAvatarId);

  const [items, setItems] = useState<ResaleListing[]>([]);
  const [imageUrlByResaleId, setImageUrlByResaleId] = useState<ResaleImageMap>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const hasItems = items.length > 0;

  const sortedItems = useMemo(() => {
    return [...items].sort((a, b) => {
      const aTime = new Date(a.updatedAt ?? a.createdAt).getTime();
      const bTime = new Date(b.updatedAt ?? b.createdAt).getTime();

      if (Number.isNaN(aTime) && Number.isNaN(bTime)) {
        return b.id.localeCompare(a.id, "ja");
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
          const resaleId = item.id;

          try {
            const images = isPublicAvatarMode
              ? await listPublicResaleConditionImages(resaleId)
              : await listMyResaleConditionImages(resaleId);

            return [resaleId, getPrimaryImageUrl(item, images)] as const;
          } catch {
            return [resaleId, ""] as const;
          }
        }),
      );

      const nextMap: ResaleImageMap = {};

      for (const [resaleId, imageUrl] of entries) {
        nextMap[resaleId] = imageUrl;
      }

      return nextMap;
    },
    [isPublicAvatarMode],
  );

  const loadResales = useCallback(async () => {
    setLoading(true);
    setError("");

    try {
      const result = isPublicAvatarMode
        ? await listResaleListingsByAvatarId({
            avatarId: normalizedAvatarId,
            page: 1,
            perPage: 50,
          })
        : await listMyResaleListings({
            page: 1,
            perPage: 50,
          });

      const nextItems = result.items;
      const nextImageMap = await loadResaleImages(nextItems);

      setItems(nextItems);
      setImageUrlByResaleId(nextImageMap);
    } catch (caught) {
      setError(
        caught instanceof Error
          ? caught.message
          : "出品一覧の取得に失敗しました。",
      );
    } finally {
      setLoading(false);
    }
  }, [
    isPublicAvatarMode,
    loadResaleImages,
    normalizedAvatarId,
  ]);

  useEffect(() => {
    void loadResales();
  }, [loadResales]);

  const handleItemClick = (item: ResaleListing) => {
    if (!onItemClick) {
      return;
    }

    onItemClick(item.id, item);
  };

  const handleItemKeyDown = (
    event: React.KeyboardEvent<HTMLElement>,
    item: ResaleListing,
  ) => {
    if (event.key !== "Enter" && event.key !== " ") {
      return;
    }

    event.preventDefault();
    handleItemClick(item);
  };

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
        const resaleId = item.id;
        const imageUrl = imageUrlByResaleId[resaleId] ?? "";
        const productName = textOrEmpty(item.productName);
        const tokenName = textOrEmpty(item.tokenName);
        const brandName = textOrEmpty(item.brandName);
        const statusLabel = formatStatusLabel(item.status);
        const statusClassName = getStatusClassName(item.status);
        const isClickable = Boolean(onItemClick);

        return (
          <article
            key={resaleId}
            className={
              isClickable
                ? "wallet-resale-list__item wallet-resale-list__item--clickable"
                : "wallet-resale-list__item"
            }
            role={isClickable ? "button" : undefined}
            tabIndex={isClickable ? 0 : undefined}
            aria-label={
              isClickable
                ? `${productName || tokenName || brandName || "出品商品"}の詳細を開く`
                : undefined
            }
            onClick={isClickable ? () => handleItemClick(item) : undefined}
            onKeyDown={
              isClickable
                ? (event) => handleItemKeyDown(event, item)
                : undefined
            }
          >
            <div className="wallet-resale-card">
              <div
                className={`wallet-resale-card__media ${statusClassName}`}
                data-status-label={statusLabel}
              >
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