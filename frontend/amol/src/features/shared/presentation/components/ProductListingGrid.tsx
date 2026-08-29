// frontend/amol/src/features/shared/presentation/components/ProductListingGrid.tsx

import ProductListingCard, {
  type ProductListingCardViewModel,
} from "./ProductListingCard";

import "../../styles/product-listing.css";

export type { ProductListingCardViewModel } from "./ProductListingCard";

export type ProductListingGridProps = {
  items: ProductListingCardViewModel[];
  onOpen: (id: string) => void;
  emptyText?: string;
  className?: string;
};

function joinClassNames(...classNames: Array<string | undefined | false>): string {
  return classNames.filter(Boolean).join(" ");
}

export default function ProductListingGrid({
  items,
  onOpen,
  emptyText = "表示できる商品がありません。",
  className,
}: ProductListingGridProps) {
  if (items.length === 0) {
    return <p className="product-listing-grid__empty">{emptyText}</p>;
  }

  return (
    <div className={joinClassNames("product-listing-grid", className)}>
      {items.map((item) => (
        <ProductListingCard
          key={item.id}
          item={item}
          onOpen={onOpen}
        />
      ))}
    </div>
  );
}