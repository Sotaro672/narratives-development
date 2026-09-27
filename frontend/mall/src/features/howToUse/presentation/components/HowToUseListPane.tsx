// frontend/mall/src/features/howToUse/presentation/components/HowToUseListPane.tsx

import { useCallback, useEffect, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";

import {
  consoleItems,
  isHowToUseCategory,
  mallItems,
  type HowToUseCategory,
  type HowToUseItem,
  type HowToUseSectionItem,
} from "../../application/howToUseSteps";

type SelectedHowToUseItem = {
  category: HowToUseCategory;
  slug: string;
} | null;

type ItemListProps = {
  title: string;
  items: HowToUseItem[];
  selectedItem: SelectedHowToUseItem;
  selectedSectionId: string | null;
  expandedItemKey: string | null;
  onItemToggle: (item: HowToUseItem) => void;
  onSectionClick: (item: HowToUseItem, section: HowToUseSectionItem) => void;
};

function getItemKey(item: Pick<HowToUseItem, "category" | "slug">): string {
  return `${item.category}-${item.slug}`;
}

function ItemList({
  title,
  items,
  selectedItem,
  selectedSectionId,
  expandedItemKey,
  onItemToggle,
  onSectionClick,
}: ItemListProps) {
  return (
    <section className="how-to-use-section">
      <h2 className="how-to-use-section__title">{title}</h2>

      <div className="how-to-use-section__items">
        {items.map((item) => {
          const itemKey = getItemKey(item);
          const selected = selectedItem?.category === item.category && selectedItem.slug === item.slug;
          const expanded = expandedItemKey === itemKey;
          const sectionListId = `how-to-use-sections-${itemKey}`;

          return (
            <div key={itemKey} className={["how-to-use-item-group", expanded ? "how-to-use-item-group--expanded" : ""].filter(Boolean).join(" ")}>
              <button
                type="button"
                className={["how-to-use-item", selected ? "how-to-use-item--selected" : "", expanded ? "how-to-use-item--expanded" : ""].filter(Boolean).join(" ")}
                aria-current={selected ? "page" : undefined}
                aria-expanded={expanded}
                aria-controls={sectionListId}
                onClick={() => {
                  onItemToggle(item);
                }}
              >
                <span className="how-to-use-item__title">{item.title}</span>
                <span className="how-to-use-item__arrow" aria-hidden="true" />
              </button>

              {expanded && (
                <div id={sectionListId} className="how-to-use-item-sections">
                  {item.sections.map((section) => {
                    const sectionSelected = selected && selectedSectionId === section.id;

                    return (
                      <button
                        key={section.id}
                        type="button"
                        className={["how-to-use-item-section", sectionSelected ? "how-to-use-item-section--selected" : ""].filter(Boolean).join(" ")}
                        aria-current={sectionSelected ? "location" : undefined}
                        onClick={() => {
                          onSectionClick(item, section);
                        }}
                      >
                        {section.title}
                      </button>
                    );
                  })}
                </div>
              )}
            </div>
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
  const selectedSectionId = getSelectedHowToUseSectionId(location.hash);
  const selectedItemKey = selectedItem ? getItemKey(selectedItem) : null;
  const [expandedItemKey, setExpandedItemKey] = useState<string | null>(selectedItemKey);

  useEffect(() => {
    if (selectedItemKey) {
      setExpandedItemKey(selectedItemKey);
    }
  }, [selectedItemKey]);

  const handleItemToggle = useCallback((item: HowToUseItem) => {
    const itemKey = getItemKey(item);
    setExpandedItemKey((current) => current === itemKey ? null : itemKey);
  }, []);

  const handleSectionClick = useCallback(
    (item: HowToUseItem, section: HowToUseSectionItem) => {
      setExpandedItemKey(getItemKey(item));
      navigate({
        pathname: `/how-to-use/${encodeURIComponent(item.category)}/${encodeURIComponent(item.slug)}`,
        hash: `#${encodeURIComponent(section.id)}`,
      });
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
          selectedSectionId={selectedSectionId}
          expandedItemKey={expandedItemKey}
          onItemToggle={handleItemToggle}
          onSectionClick={handleSectionClick}
        />

        <ItemList
          title="購入者 Mall"
          items={mallItems}
          selectedItem={selectedItem}
          selectedSectionId={selectedSectionId}
          expandedItemKey={expandedItemKey}
          onItemToggle={handleItemToggle}
          onSectionClick={handleSectionClick}
        />
      </div>
    </main>
  );
}

function getSelectedHowToUseItem(pathname: string): SelectedHowToUseItem {
  const segments = pathname.split("/").filter(Boolean).map(decodePathSegment);

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

function getSelectedHowToUseSectionId(hash: string): string | null {
  const value = hash.startsWith("#") ? hash.slice(1) : hash;

  if (!value) {
    return null;
  }

  return decodePathSegment(value);
}

function decodePathSegment(value: string): string {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}