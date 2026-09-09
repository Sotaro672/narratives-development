// frontend/admin/shell/src/pages/ListDetailPage.tsx

import { useMemo } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { useContractListDetail } from "../features/company/presentation/hooks/useContractListDetail";
import type { ContractListPriceRow } from "../shared/type/contractListDetail";
import MediaGallery, { type MediaGalleryItem } from "../shared/ui/MediaGallery/MediaGallery";
import Page, { DetailPageBody, PageHeader } from "../shared/ui/Page/Page";
import { formatDateTime } from "../shared/util/dateFormat";

function formatListStatus(status: string): string {
  switch (status) {
    case "listing":
      return "出品中";
    case "suspended":
      return "停止中";
    default:
      return status || "-";
  }
}

function formatModelMeta(price: ContractListPriceRow): string {
  const values: string[] = [];

  if (price.modelNumber) {
    values.push(price.modelNumber);
  }

  if (price.kind === "apparel") {
    if (price.size) {
      values.push(price.size);
    }
    if (price.color) {
      values.push(price.color);
    }
  }

  if (price.kind === "alcohol" && price.volumeValue != null) {
    values.push(`${price.volumeValue}${price.volumeUnit || ""}`);
  }

  return values.length > 0 ? values.join(" / ") : "モデル情報なし";
}

export default function ListDetailPage() {
  const navigate = useNavigate();
  const { companyId = "", listId = "" } = useParams<{
    companyId?: string;
    listId?: string;
  }>();
  const { detail, loading, error, reload } = useContractListDetail(companyId, listId);
  const list = detail?.list ?? null;

  const galleryItems = useMemo<MediaGalleryItem[]>(() => {
    if (!list) {
      return [];
    }

    const images = Array.isArray(list.images) ? list.images : [];

    return [...images]
      .filter((image) => Boolean(image.id && image.url))
      .sort((a, b) => {
        if (a.displayOrder !== b.displayOrder) {
          return a.displayOrder - b.displayOrder;
        }
        return a.id.localeCompare(b.id);
      })
      .map((image) => ({
        id: image.id,
        url: image.url,
      }));
  }, [list]);

  const renderMain = () => {
    if (loading && !detail) {
      return <p>出品詳細を取得しています...</p>;
    }

    if (error && !detail) {
      return (
        <div role="alert">
          <p>出品詳細を取得できませんでした。</p>
          <p>{error}</p>
          <button type="button" onClick={() => void reload()}>
            再読み込み
          </button>
        </div>
      );
    }

    if (!list) {
      return <p role="alert">出品情報を取得できませんでした。</p>;
    }

    return (
      <>
        <section className="ui-detail-section">
          <MediaGallery
            items={galleryItems}
            altFallback={list.title || list.productName || "出品画像"}
            placeholderText="出品画像はありません。"
          />
        </section>

        <section className="ui-detail-section">
          <p className="ui-detail-section__text">{list.description || "-"}</p>
        </section>

        <section className="ui-detail-section">
          <h2 className="ui-detail-section__title">価格</h2>
          {list.prices.length > 0 ? (
            <dl className="ui-detail-definition-list">
              {list.prices.map((price) => (
                <div className="ui-detail-price-row" key={price.modelId}>
                  <dt>{formatModelMeta(price)}</dt>
                  <dd>{price.price.toLocaleString("ja-JP")}円</dd>
                </div>
              ))}
            </dl>
          ) : (
            <p>価格情報はありません。</p>
          )}
        </section>
      </>
    );
  };

  return (
    <Page>
      <PageHeader
        title={list?.title || list?.readableId || "出品詳細"}
        leading={
          <button
            type="button"
            className="ui-page-header__back"
            aria-label="戻る"
            onClick={() =>
              navigate(
                companyId
                  ? `/contracts/${encodeURIComponent(companyId)}`
                  : "/contracts",
              )
            }
          >
            <svg
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              aria-hidden="true"
            >
              <path d="M19 12H5" />
              <path d="M12 19l-7-7 7-7" />
            </svg>
          </button>
        }
      />

      {list ? (
        <DetailPageBody
          main={renderMain()}
          aside={
            <>
              <section className="ui-detail-section">
                <dl className="ui-detail-definition-list">
                  <dt>商品名</dt>
                  <dd>{list.productName || "-"}</dd>

                  <dt>商品ブランド名</dt>
                  <dd>{list.productBrandName || "-"}</dd>
                </dl>
              </section>

              <section className="ui-detail-section">
                <dl className="ui-detail-definition-list">
                  <dt>トークン名</dt>
                  <dd>{list.tokenName || "-"}</dd>

                  <dt>トークンブランド名</dt>
                  <dd>{list.tokenBrandName || "-"}</dd>
                </dl>
              </section>

              <section className="ui-detail-section">
                <dl className="ui-detail-definition-list">
                  <dt>出品ID</dt>
                  <dd>{list.readableId || list.id || "-"}</dd>

                  <dt>出品状態</dt>
                  <dd>{formatListStatus(list.status)}</dd>

                  <dt>担当者</dt>
                  <dd>{list.assigneeName || "-"}</dd>

                  <dt>作成日時</dt>
                  <dd className="ui-detail-definition-list__nowrap">
                    {formatDateTime(list.createdAt)}
                  </dd>

                  <dt>最終更新日時</dt>
                  <dd className="ui-detail-definition-list__nowrap">
                    {formatDateTime(list.updatedAt)}
                  </dd>
                </dl>
              </section>
            </>
          }
        />
      ) : (
        renderMain()
      )}
    </Page>
  );
}