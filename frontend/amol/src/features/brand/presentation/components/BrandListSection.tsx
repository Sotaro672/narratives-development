// frontend/amol/src/features/brand/presentation/components/BrandListSection.tsx

import { Link } from "react-router-dom";
import type { BrandListItem } from "../../types/brand";
import { formatBrandListPrice } from "../utils/formatBrandListPrice";

type BrandListSectionProps = {
  listIds: string[];
  listItems: BrandListItem[];
};

export default function BrandListSection({
  listIds,
  listItems,
}: BrandListSectionProps) {
  if (listIds.length === 0) {
    return (
      <section className="brand-page-section">
        <h2>出品中のリスト</h2>

        <div className="brand-page-empty">
          現在このブランドの出品中リストはありません。
        </div>
      </section>
    );
  }

  if (listItems.length === 0) {
    return (
      <section className="brand-page-section">
        <div className="brand-page-section-header">
          <h2>出品中のリスト</h2>
          <span>{listIds.length}件</span>
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
        <h2>出品中のリスト</h2>
        <span>{listItems.length}件</span>
      </div>

      <div className="lists-page-grid brand-page-list-grid">
        {listItems.map((item) => (
          <Link
            key={item.id}
            className="lists-page-card brand-page-list-card"
            to={`/lists/${encodeURIComponent(item.id)}`}
          >
            <div className="lists-page-card-image-wrap">
              {item.image ? (
                <img
                  src={item.image}
                  alt={item.title}
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
              <h2 className="lists-page-card-title">{item.title}</h2>

              {item.description ? (
                <p className="lists-page-card-description">
                  {item.description}
                </p>
              ) : null}

              <div className="lists-page-card-footer">
                <span className="lists-page-card-price">
                  {formatBrandListPrice(item.prices)}
                </span>
              </div>
            </div>
          </Link>
        ))}
      </div>
    </section>
  );
}