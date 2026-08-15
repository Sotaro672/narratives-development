// frontend/amol/src/features/list/presentation/components/ListGrid.tsx

import ListCard from "./ListCard";

import type {
  MallListCardItem,
} from "../../../shared/types/list";

type ListGridProps = {
  items: MallListCardItem[];
  onOpenItem: (
    listId: string,
  ) => void;
};

export default function ListGrid({
  items,
  onOpenItem,
}: ListGridProps) {
  if (items.length === 0) {
    return null;
  }

  return (
    <div className="lists-page-grid">
      {items.map((item) => (
        <ListCard
          key={item.id}
          item={item}
          onOpenItem={
            onOpenItem
          }
        />
      ))}
    </div>
  );
}