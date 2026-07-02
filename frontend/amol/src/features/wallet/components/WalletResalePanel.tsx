// frontend/amol/src/features/wallet/components/WalletResalePanel.tsx
import { useCallback, useEffect, useMemo, useState } from "react";

import { listMyResaleListings } from "../../resale/api/resaleApi";
import type { ResaleListing } from "../../resale/api/resaleApi";

function formatPrice(value: number | undefined): string {
  const price = Number(value ?? 0);

  if (!Number.isFinite(price) || price <= 0) {
    return "-";
  }

  return `¥${price.toLocaleString("ja-JP")}`;
}

function formatStatus(value: string | undefined): string {
  switch (value) {
    case "listing":
      return "出品中";
    case "suspended":
      return "停止中";
    default:
      return value || "-";
  }
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

function getDisplayTitle(item: ResaleListing): string {
  if (item.productId) {
    return `商品ID: ${item.productId}`;
  }

  if (item.mintAddress) {
    return `Mint: ${item.mintAddress.slice(0, 8)}...`;
  }

  return "出品商品";
}

export default function WalletResalePanel() {
  const [items, setItems] = useState<ResaleListing[]>([]);
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

  const loadResales = useCallback(async () => {
    setLoading(true);
    setError("");

    try {
      const result = await listMyResaleListings({
        page: 1,
        perPage: 50,
      });

      setItems(result.items ?? []);
    } catch (error) {
      setError(
        error instanceof Error
          ? error.message
          : "出品一覧の取得に失敗しました。",
      );
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadResales();
  }, [loadResales]);

  if (loading) {
    return (
      <div className="wallet-page-token-list">
        <p className="wallet-page__message">読み込み中です...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="wallet-page-token-list">
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
      <div className="wallet-page-token-list">
        <div className="wallet-page__message">
          <p>出品中の商品はありません。</p>
        </div>
      </div>
    );
  }

  return (
    <div className="wallet-page-token-list">
      {sortedItems.map((item) => (
        <article
          key={item.id || item.productId || item.mintAddress}
          className="wallet-page-token-list__item"
        >
          <div className="wallet-resale-card">
            <div className="wallet-resale-card__body">
              <div className="wallet-resale-card__header">
                <p className="wallet-resale-card__title">
                  {getDisplayTitle(item)}
                </p>

                <span className="wallet-resale-card__status">
                  {formatStatus(item.status)}
                </span>
              </div>

              <dl className="wallet-resale-card__meta">
                <div className="wallet-resale-card__meta-row">
                  <dt>価格</dt>
                  <dd>{formatPrice(item.price)}</dd>
                </div>

                <div className="wallet-resale-card__meta-row">
                  <dt>状態</dt>
                  <dd>{item.condition || "-"}</dd>
                </div>

                <div className="wallet-resale-card__meta-row">
                  <dt>出品日</dt>
                  <dd>{formatDateTime(item.createdAt)}</dd>
                </div>

                <div className="wallet-resale-card__meta-row">
                  <dt>画像</dt>
                  <dd>{item.imageId ? "設定済み" : "未設定"}</dd>
                </div>
              </dl>

              {item.description ? (
                <p className="wallet-resale-card__description">
                  {item.description}
                </p>
              ) : null}
            </div>
          </div>
        </article>
      ))}
    </div>
  );
}