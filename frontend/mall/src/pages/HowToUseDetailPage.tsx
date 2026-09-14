// frontend/mall/src/pages/HowToUseDetailPage.tsx

import { Navigate, useParams } from "react-router-dom";

import Layout from "../components/layout/Layout";
import {
  findHowToUseItem,
  isHowToUseCategory,
  type HowToUseCategory,
} from "../features/howToUse/application/howToUseSteps";
import HowToUseArticle from "../features/howToUse/presentation/components/common/HowToUseArticle";
import BrandRegistrationGuide from "../features/howToUse/presentation/components/console/BrandRegistrationGuide";
import MemberInvitationGuide from "../features/howToUse/presentation/components/console/MemberInvitationGuide";
import ProductBlueprintGuide from "../features/howToUse/presentation/components/console/ProductBlueprintGuide";

import "../styles/page-layout.css";
import "../styles/how-to-use-detail-page.css";
import "../styles/how-to-use-common.css";

function renderGuide(category: HowToUseCategory, slug: string) {
  if (category === "console" && slug === "brand-registration") {
    return <BrandRegistrationGuide />;
  }

  if (category === "console" && slug === "member-invite") {
    return <MemberInvitationGuide />;
  }

  if (category === "console" && slug === "product-design") {
    return <ProductBlueprintGuide />;
  }

  return null;
}

export default function HowToUseDetailPage() {
  const { category, slug = "" } = useParams<{
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
          <HowToUseArticle description={item.description}>
            {renderGuide(category, slug)}
          </HowToUseArticle>
        </div>
      </main>
    </Layout>
  );
}