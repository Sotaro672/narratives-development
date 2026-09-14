// frontend/mall/src/pages/HowToUseDetailPage.tsx

import { Navigate, useParams } from "react-router-dom";

import Layout from "../components/layout/Layout";
import {
  findHowToUseItem,
  isHowToUseCategory,
} from "../features/howToUse/application/howToUseSteps";

import "../styles/page-layout.css";
import "../styles/how-to-use-detail-page.css";

export default function HowToUseDetailPage() {
  const {
    category,
    slug = "",
  } = useParams<{
    category?: string;
    slug?: string;
  }>();

  if (!isHowToUseCategory(category) || !slug) {
    return <Navigate to="/how-to-use" replace />;
  }

  const item = findHowToUseItem(category, slug);

  if (!item) {
    return <Navigate to="/how-to-use" replace />;
  }

  return (
    <Layout
      title={item.title}
      titleClickable={false}
      mode="landing"
      showBackButton
      backTo="/how-to-use"
      hideAnnouncementButton
      hideSettingsButton
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