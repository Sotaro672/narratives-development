// frontend/amol/src/features/shared/presentation/components/ProductPrice.tsx

export type ProductPriceProps = {
  priceLabel?: string | null;
  className?: string;
};

function joinClassNames(...classNames: Array<string | undefined | false>): string {
  return classNames.filter(Boolean).join(" ");
}

export default function ProductPrice({
  priceLabel,
  className,
}: ProductPriceProps) {
  const safePriceLabel = priceLabel?.trim() || "-";

  return (
    <p className={joinClassNames("product-detail__price", className)}>
      {safePriceLabel}
    </p>
  );
}