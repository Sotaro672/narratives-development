// frontend/mall/src/pages/HowToUseDetailPage.tsx

import { Navigate, useParams } from "react-router-dom";

import Layout from "../components/layout/Layout";

import {
  buyerItems,
  sellerItems,
  type HowToUseCategory,
  type HowToUseItem,
} from "./HowToUsePage";

import "../styles/page-layout.css";
import "../styles/how-to-use-detail-page.css";

function isHowToUseCategory(
  value: string | undefined,
): value is HowToUseCategory {
  return value === "seller" || value === "buyer";
}

function findItem(
  category: HowToUseCategory,
  slug: string,
): HowToUseItem | undefined {
  const items =
    category === "seller"
      ? sellerItems
      : buyerItems;

  return items.find((item) => item.slug === slug);
}

export default function HowToUseDetailPage() {
  const {
    category,
    slug = "",
  } = useParams<{
    category?: string;
    slug?: string;
  }>();

  if (
    !isHowToUseCategory(category) ||
    !slug
  ) {
    return (
      <Navigate
        to="/how-to-use"
        replace
      />
    );
  }

  const item = findItem(category, slug);

  if (!item) {
    return (
      <Navigate
        to="/how-to-use"
        replace
      />
    );
  }

  return (
    <Layout
      title={item.title}
      mode="landing"
      showBackButton
      backTo="/how-to-use"
    >
      <main className="how-to-use-detail-page">
        <div className="how-to-use-detail-page__inner">
          <p className="how-to-use-detail-page__description">
            {item.description}
          </p>
        </div>
      </main>
    </Layout>
  );
}