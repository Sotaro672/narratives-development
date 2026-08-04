// frontend/amol/src/features/brand/presentation/components/BrandListSection.tsx

import {
  Link,
} from "react-router-dom";

import type {
  BrandListItem,
} from "../../types/brand";

import {
  formatBrandListPrice,
} from "../utils/formatBrandListPrice";

type BrandListSectionProps = {
  listIds: string[];
  listItems: BrandListItem[];
};

export default function BrandListSection({
  listIds,
  listItems,
}: BrandListSectionProps) {
  const normalizedListIds =
    Array.isArray(listIds)
      ? listIds
          .map((listId) =>
            listId.trim(),
          )
          .filter(Boolean)
      : [];

  const validListItems =
    Array.isArray(listItems)
      ? listItems.filter(
          (item) =>
            item.id.trim().length > 0,
        )
      : [];

  if (
    normalizedListIds.length === 0
  ) {
    return (
      <section className="brand-page-section">
        <h2>
          出品中のリスト
        </h2>

        <div className="brand-page-empty">
          現在このブランドの出品中リストはありません。
        </div>
      </section>
    );
  }

  if (
    validListItems.length === 0
  ) {
    return (
      <section className="brand-page-section">
        <div className="brand-page-section-header">
          <h2>
            出品中のリスト
          </h2>

          <span>
            {normalizedListIds.length}件
          </span>
        </div>

        <div className="brand-page-empty">
          リスト情報を取得できませんでした。
        </div>
      </section>
    );
  }

  return (
    <section className="brand-page-section">
      <div className="brand-page-section-header">
        <h2>
          出品中のリスト
        </h2>

        <span>
          {validListItems.length}件
        </span>
      </div>

      <div className="lists-page-grid brand-page-list-grid">
        {validListItems.map(
          (item) => {
            const itemId =
              item.id.trim();

            const title =
              item.title.trim() ||
              itemId;

            const description =
              item.description.trim();

            const image =
              item.image.trim();

            return (
              <Link
                key={itemId}
                className="lists-page-card brand-page-list-card"
                to={`/lists/${encodeURIComponent(
                  itemId,
                )}`}
              >
                <div className="lists-page-card-image-wrap">
                  {image ? (
                    <img
                      src={image}
                      alt={title}
                      className="lists-page-card-image"
                      loading="lazy"
                    />
                  ) : (
                    <div className="lists-page-card-image-placeholder">
                      No Image
                    </div>
                  )}
                </div>

                <div className="lists-page-card-body">
                  <h2 className="lists-page-card-title">
                    {title}
                  </h2>

                  {description ? (
                    <p className="lists-page-card-description">
                      {description}
                    </p>
                  ) : null}

                  <div className="lists-page-card-footer">
                    <span className="lists-page-card-price">
                      {formatBrandListPrice(
                        item.prices,
                      )}
                    </span>
                  </div>
                </div>
              </Link>
            );
          },
        )}
      </div>
    </section>
  );
}