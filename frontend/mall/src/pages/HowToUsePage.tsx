// frontend/mall/src/pages/HowToUsePage.tsx

import { useNavigate } from "react-router-dom";

import Layout from "../components/layout/Layout";
import Button from "../components/ui/Button";
import {
  consoleItems,
  mallItems,
  type HowToUseItem,
} from "../features/howToUse/application/howToUseSteps";

import "../styles/page-layout.css";
import "../styles/how-to-use-page.css";

type ItemListProps = {
  title: string;
  items: HowToUseItem[];
  onDetailClick: (item: HowToUseItem) => void;
};

function ItemList({ title, items, onDetailClick }: ItemListProps) {
  return (
    <section className="how-to-use-section">
      <h2 className="how-to-use-section__title">{title}</h2>

      <div className="how-to-use-section__items">
        {items.map((item) => (
          <details
            key={`${item.category}-${item.slug}`}
            className="how-to-use-item"
          >
            <summary className="how-to-use-item__summary">
              <span className="how-to-use-item__title">{item.title}</span>
              <span
                className="how-to-use-item__arrow"
                aria-hidden="true"
              />
            </summary>

            <div className="how-to-use-item__content">
              <p className="how-to-use-item__description">
                {item.description}
              </p>

              <button
                type="button"
                className="how-to-use-item__detail-link"
                onClick={() => onDetailClick(item)}
              >
                詳しく見る
              </button>
            </div>
          </details>
        ))}
      </div>
    </section>
  );
}

export default function HowToUsePage() {
  const navigate = useNavigate();

  const handleDetailClick = (item: HowToUseItem) => {
    navigate(`/how-to-use/${item.category}/${item.slug}`);
  };

  return (
    <Layout
      title="AMOL"
      mode="landing"
      hideAnnouncementButton
    >
      <main className="how-to-use-page">
        <div className="how-to-use-page__inner">
          <ItemList
            title="出品者Consoleの使い方"
            items={consoleItems}
            onDetailClick={handleDetailClick}
          />

          <ItemList
            title="購入者Mallの使い方"
            items={mallItems}
            onDetailClick={handleDetailClick}
          />

          <div className="page-actions">
            <Button
              variant="primary"
              onClick={() => navigate("/signin/select")}
            >
              試作品を体験
            </Button>
          </div>
        </div>
      </main>
    </Layout>
  );
}