// frontend/amol/src/features/shared/presentation/components/ProductListingCard.tsx

import "../../styles/product-listing.css";

export type ProductListingCardViewModel = {
  id: string;
  title: string;
  imageUrl?: string | null;
  imageAlt?: string | null;
  brandName?: string | null;
  metaLines?: Array<string | null | undefined>;
  priceLabel?: string | null;
  reviewAverage?: number | null;
  reviewCount?: number | null;
};

export type ProductListingCardProps = {
  item: ProductListingCardViewModel;
  onOpen: (id: string) => void;
  className?: string;
};

function joinClassNames(...classNames: Array<string | undefined | false>): string {
  return classNames.filter(Boolean).join(" ");
}

export default function ProductListingCard({
  item,
  onOpen,
  className,
}: ProductListingCardProps) {
  const safeId = item.id.trim();
  const safeTitle = item.title.trim() || "商品名未設定";
  const safeImageUrl = item.imageUrl?.trim() || "";
  const safeImageAlt = item.imageAlt?.trim() || safeTitle;
  const safeBrandName = item.brandName?.trim() || "";
  const safeMetaLines = (item.metaLines ?? [])
    .map((line) => line?.trim() || "")
    .filter(Boolean);
  const safePriceLabel = item.priceLabel?.trim() || "";

  const safeReviewAverage =
    typeof item.reviewAverage === "number" && Number.isFinite(item.reviewAverage)
      ? Math.max(0, Math.min(5, item.reviewAverage))
      : null;

  const safeReviewCount =
    typeof item.reviewCount === "number" && Number.isFinite(item.reviewCount)
      ? Math.max(0, Math.floor(item.reviewCount))
      : null;

  const reviewLabel =
    safeReviewCount === 0
      ? "レビューなし"
      : safeReviewCount !== null && safeReviewAverage !== null
        ? `★ ${safeReviewAverage.toFixed(1)} (${safeReviewCount})`
        : null;

  function handleClick() {
    if (!safeId) return;
    onOpen(safeId);
  }

  return (
    <button
      type="button"
      className={joinClassNames("product-listing-card", className)}
      onClick={handleClick}
      disabled={!safeId}
    >
      <div className="product-listing-card__image-wrap">
        {safeImageUrl ? (
          <img
            src={safeImageUrl}
            alt={safeImageAlt}
            className="product-listing-card__image"
            loading="lazy"
          />
        ) : (
          <div className="product-listing-card__image-placeholder">No Image</div>
        )}
      </div>

      <div className="product-listing-card__body">
        <h2 className="product-listing-card__title">{safeTitle}</h2>

        {safeBrandName ? (
          <p className="product-listing-card__brand">{safeBrandName}</p>
        ) : null}

        {reviewLabel ? (
          <p className="product-listing-card__review">{reviewLabel}</p>
        ) : null}

        {safeMetaLines.length > 0 ? (
          <div className="product-listing-card__meta">
            {safeMetaLines.map((line, index) => (
              <p key={`${safeId}-meta-${index}`} className="product-listing-card__meta-line">
                {line}
              </p>
            ))}
          </div>
        ) : null}

        {safePriceLabel ? (
          <div className="product-listing-card__footer">
            <span className="product-listing-card__price">{safePriceLabel}</span>
          </div>
        ) : null}
      </div>
    </button>
  );
}