// frontend/mall/src/features/howToUse/presentation/components/HowToUseListPane.tsx

import { useCallback } from "react";
import { useLocation, useNavigate } from "react-router-dom";

import {
  consoleItems,
  isHowToUseCategory,
  mallItems,
  type HowToUseCategory,
  type HowToUseItem,
} from "../../application/howToUseSteps";

type SelectedHowToUseItem = {
  category: HowToUseCategory;
  slug: string;
} | null;

type ItemListProps = {
  title: string;
  items: HowToUseItem[];
  selectedItem: SelectedHowToUseItem;
  onDetailClick: (item: HowToUseItem) => void;
};

function ItemList({
  title,
  items,
  selectedItem,
  onDetailClick,
}: ItemListProps) {
  return (
    <section className="how-to-use-section">
      <h2 className="how-to-use-section__title">{title}</h2>

      <div className="how-to-use-section__items">
        {items.map((item) => {
          const selected =
            selectedItem?.category === item.category &&
            selectedItem.slug === item.slug;

          return (
            <button
              key={`${item.category}-${item.slug}`}
              type="button"
              className={[
                "how-to-use-item",
                selected ? "how-to-use-item--selected" : "",
              ].filter(Boolean).join(" ")}
              aria-current={selected ? "page" : undefined}
              onClick={() => {
                onDetailClick(item);
              }}
            >
              <span className="how-to-use-item__title">{item.title}</span>
              <span className="how-to-use-item__arrow" aria-hidden="true" />
            </button>
          );
        })}
      </div>
    </section>
  );
}

export default function HowToUseListPane() {
  const location = useLocation();
  const navigate = useNavigate();

  const selectedItem = getSelectedHowToUseItem(location.pathname);

  const handleDetailClick = useCallback(
    (item: HowToUseItem) => {
      navigate(
        `/how-to-use/${encodeURIComponent(item.category)}/${encodeURIComponent(item.slug)}`,
      );
    },
    [navigate],
  );

  return (
    <main className="how-to-use-page how-to-use-page--workspace">
      <div className="how-to-use-page__inner">
        <ItemList
          title="出品者 Console"
          items={consoleItems}
          selectedItem={selectedItem}
          onDetailClick={handleDetailClick}
        />

        <ItemList
          title="購入者 Mall"
          items={mallItems}
          selectedItem={selectedItem}
          onDetailClick={handleDetailClick}
        />
      </div>
    </main>
  );
}

function getSelectedHowToUseItem(pathname: string): SelectedHowToUseItem {
  const segments = pathname
    .split("/")
    .filter(Boolean)
    .map(decodePathSegment);

  if (segments[0] !== "how-to-use" || segments.length !== 3) {
    return null;
  }

  const category = segments[1];
  const slug = segments[2];

  if (!isHowToUseCategory(category) || !slug) {
    return null;
  }

  return {
    category,
    slug,
  };
}

function decodePathSegment(value: string): string {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}