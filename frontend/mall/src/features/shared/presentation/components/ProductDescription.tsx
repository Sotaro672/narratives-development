// frontend/amol/src/features/shared/presentation/components/ProductDescription.tsx

import {
  useId,
  useState,
} from "react";

const DESCRIPTION_COLLAPSE_THRESHOLD = 80;

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
  const descriptionId = useId();
  const [descriptionExpanded, setDescriptionExpanded] = useState(false);

  const safeDescription = description?.trim() || "";
  const isDescriptionExpandable = safeDescription.length > DESCRIPTION_COLLAPSE_THRESHOLD;

  if (!safeDescription) {
    return null;
  }

  return (
    <div className={joinClassNames("product-detail__description", className)}>
      <h2>{title}</h2>

      <p
        id={descriptionId}
        className={joinClassNames(
          "product-detail__description-text",
          isDescriptionExpandable &&
            !descriptionExpanded &&
            "product-detail__description-text--collapsed",
        )}
      >
        {safeDescription}
      </p>

      {isDescriptionExpandable ? (
        <button
          type="button"
          className="product-detail__description-toggle"
          aria-expanded={descriptionExpanded}
          aria-controls={descriptionId}
          onClick={() => setDescriptionExpanded((current) => !current)}
        >
          {descriptionExpanded ? "閉じる" : "詳しく見る"}
        </button>
      ) : null}
    </div>
  );
}