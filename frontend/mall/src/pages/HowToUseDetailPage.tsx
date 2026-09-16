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
import InquiryGuide from "../features/howToUse/presentation/components/console/InquiryGuide";
import InspectionGuide from "../features/howToUse/presentation/components/console/InspectionGuide";
import ListGuide from "../features/howToUse/presentation/components/console/ListGuide";
import MemberInvitationGuide from "../features/howToUse/presentation/components/console/MemberInvitationGuide";
import MintGuide from "../features/howToUse/presentation/components/console/MintGuide";
import OrderDispatchGuide from "../features/howToUse/presentation/components/console/OrderDispatchGuide";
import ProductBlueprintGuide from "../features/howToUse/presentation/components/console/ProductBlueprintGuide";
import ProductionRegistrationGuide from "../features/howToUse/presentation/components/console/ProductionRegistrationGuide";
import SetLocationGuide from "../features/howToUse/presentation/components/console/SetLocationGuide";
import SetTranspportationFee from "../features/howToUse/presentation/components/console/SetTranspportationFee";
import TokenBlueprintGuide from "../features/howToUse/presentation/components/console/TokenBlueprintGuide";
import AvatarCreateGuide from "../features/howToUse/presentation/components/mall/AvatarCreateGuide";
import CancelOderGuide from "../features/howToUse/presentation/components/mall/CancelOderGuide";
import PurchaseGuide from "../features/howToUse/presentation/components/mall/PurchaseGuide";
import RequestRefundGuide from "../features/howToUse/presentation/components/mall/RequestRefundGuide";
import ShippingAddressRegistrationGuide from "../features/howToUse/presentation/components/mall/ShippingAddressRegistrationGuide";

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

  if (category === "console" && slug === "token-design") {
    return <TokenBlueprintGuide />;
  }

  if (category === "console" && slug === "production") {
    return <ProductionRegistrationGuide />;
  }

  if (category === "console" && slug === "inspection") {
    return <InspectionGuide />;
  }

  if (category === "console" && slug === "mint") {
    return <MintGuide />;
  }

  if (category === "console" && slug === "inventory") {
    return <SetLocationGuide />;
  }

  if (category === "console" && slug === "shipping") {
    return <SetTranspportationFee />;
  }

  if (category === "console" && slug === "listing") {
    return <ListGuide />;
  }

  if (category === "console" && slug === "orders") {
    return <OrderDispatchGuide />;
  }

  if (category === "console" && slug === "inquiry") {
    return <InquiryGuide />;
  }

  if (category === "mall" && slug === "avatar-registration") {
    return <AvatarCreateGuide />;
  }

  if (category === "mall" && slug === "shipping-address") {
    return <ShippingAddressRegistrationGuide />;
  }

  if (category === "mall" && slug === "purchase") {
    return <PurchaseGuide />;
  }

  if (category === "mall" && slug === "cancel") {
    return <CancelOderGuide />;
  }

  if (category === "mall" && slug === "return") {
    return <RequestRefundGuide />;
  }

  return null;
}

export default function HowToUseDetailPage() {
  const { category, slug = "" } = useParams<{ category?: string; slug?: string }>();

  if (!isHowToUseCategory(category) || !slug) {
    return <Navigate to="/how-to-use" replace />;
  }

  const item = findHowToUseItem(category, slug);

  if (!item) {
    return <Navigate to="/how-to-use" replace />;
  }

  return (
    <Layout title={item.title} titleClickable={false} mode="landing" showBackButton backTo="/how-to-use" hideAnnouncementButton hideSettingsButton>
      <main className="how-to-use-detail-page">
        <div className="how-to-use-detail-page__inner">
          <HowToUseArticle description={item.description}>{renderGuide(category, slug)}</HowToUseArticle>
        </div>
      </main>
    </Layout>
  );
}