// frontend/amol/src/features/shared/presentation/components/ProductMetaList.tsx

import type { ReactNode } from "react";

export type ProductMetaListItem = {
  label: string;
  value: ReactNode;
};

export type ProductMetaListProps = {
  items: ProductMetaListItem[];
  className?: string;
};

function joinClassNames(...classNames: Array<string | undefined | false>): string {
  return classNames.filter(Boolean).join(" ");
}

export default function ProductMetaList({
  items,
  className,
}: ProductMetaListProps) {
  if (items.length === 0) {
    return null;
  }

  return (
    <dl className={joinClassNames("product-detail__meta", className)}>
      {items.map((item, index) => (
        <div
          key={`${item.label}-${index}`}
          className="product-detail__meta-row"
        >
          <dt>{item.label}</dt>
          <dd>{item.value}</dd>
        </div>
      ))}
    </dl>
  );
}