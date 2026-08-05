// frontend/amol/src/features/catalog/presentation/components/CatalogSummary.tsx

import {
  formatPrice,
} from "../../../../components/utils/price";

type CatalogSummaryProps = {
  title: string;
  description?: string | null;
  price?: number | null;
};

export default function CatalogSummary({
  title,
  description,
  price,
}: CatalogSummaryProps) {
  return (
    <div className="catalog-page-summary">
      <h1 className="catalog-page-title">
        {title}
      </h1>

      {description ? (
        <p className="catalog-page-description">
          {description}
        </p>
      ) : null}

      <p className="catalog-page-price">
        {formatPrice(price)}
      </p>
    </div>
  );
}