// frontend/amol/src/features/shared/presentation/components/ProductMetaList.tsx

import type { ReactNode } from "react";

import InfoList, { InfoRow } from "../../../../components/ui/InfoList";

export type ProductMetaListItem = {
  label: string;
  value: ReactNode;
};

export type ProductMetaListProps = {
  items: ProductMetaListItem[];
  className?: string;
};

export default function ProductMetaList({
  items,
  className,
}: ProductMetaListProps) {
  if (items.length === 0) {
    return null;
  }

  return (
    <InfoList className={className}>
      {items.map((item, index) => (
        <InfoRow key={`${item.label}-${index}`} label={item.label}>
          {item.value}
        </InfoRow>
      ))}
    </InfoList>
  );
}