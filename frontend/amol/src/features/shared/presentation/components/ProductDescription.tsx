// frontend/amol/src/features/shared/presentation/components/ProductDescription.tsx

export type ProductDescriptionProps = {
  description?: string | null;
  title?: string;
  className?: string;
};

function joinClassNames(...classNames: Array<string | undefined | false>): string {
  return classNames.filter(Boolean).join(" ");
}

export default function ProductDescription({
  description,
  title = "商品説明",
  className,
}: ProductDescriptionProps) {
  const safeDescription = description?.trim() || "";

  if (!safeDescription) {
    return null;
  }

  return (
    <div className={joinClassNames("product-detail__description", className)}>
      <h2>{title}</h2>
      <p>{safeDescription}</p>
    </div>
  );
}