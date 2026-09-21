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
import BroadcastAnnounceGuide from "../features/howToUse/presentation/components/console/BroadcastAnnounceGuide";
import InquiryGuide from "../features/howToUse/presentation/components/console/InquiryGuide";
import InspectionGuide from "../features/howToUse/presentation/components/console/InspectionGuide";
import ListGuide from "../features/howToUse/presentation/components/console/ListGuide";
import MemberInvitationGuide from "../features/howToUse/presentation/components/console/MemberInvitationGuide";
import MintGuide from "../features/howToUse/presentation/components/console/MintGuide";
import OrderDispatchGuide from "../features/howToUse/presentation/components/console/OrderDispatchGuide";
import ProductBlueprintGuide from "../features/howToUse/presentation/components/console/ProductBlueprintGuide";
import ProductionRegistrationGuide from "../features/howToUse/presentation/components/console/ProductionRegistrationGuide";
import ReplyTokenCommentGuide from "../features/howToUse/presentation/components/console/ReplyTokenCommentGuide";
import SetLocationGuide from "../features/howToUse/presentation/components/console/SetLocationGuide";
import SetTranspportationFee from "../features/howToUse/presentation/components/console/SetTranspportationFee";
import TokenBlueprintGuide from "../features/howToUse/presentation/components/console/TokenBlueprintGuide";
import ViewProductReviewGuide from "../features/howToUse/presentation/components/console/ViewProductReviewGuide";
import AvatarCreateGuide from "../features/howToUse/presentation/components/mall/AvatarCreateGuide";
import CancelOderGuide from "../features/howToUse/presentation/components/mall/CancelOderGuide";
import ListMarketGuide from "../features/howToUse/presentation/components/mall/ListMarketGuide";
import OpenPayoutAccountGuide from "../features/howToUse/presentation/components/mall/OpenPayoutAccountGuide";
import PostCommentGuide from "../features/howToUse/presentation/components/mall/PostCommentGuide";
import PostProductReviewGuide from "../features/howToUse/presentation/components/mall/PostProductReviewGuide";
import PurchaseGuide from "../features/howToUse/presentation/components/mall/PurchaseGuide";
import RequestRefundGuide from "../features/howToUse/presentation/components/mall/RequestRefundGuide";
import ReviewResaleGuide from "../features/howToUse/presentation/components/mall/ReviewResaleGuide";
import ShippingAddressRegistrationGuide from "../features/howToUse/presentation/components/mall/ShippingAddressRegistrationGuide";
import TradeGuide from "../features/howToUse/presentation/components/mall/TradeGuide";

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

  if (category === "console" && slug === "comment-reply") {
    return <ReplyTokenCommentGuide />;
  }

  if (category === "console" && slug === "reviews") {
    return <ViewProductReviewGuide />;
  }

  if (category === "console" && slug === "announcement") {
    return <BroadcastAnnounceGuide />;
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

  if (category === "mall" && slug === "comment") {
    return <PostCommentGuide />;
  }

  if (category === "mall" && slug === "cancel") {
    return <CancelOderGuide />;
  }

  if (category === "mall" && slug === "return") {
    return <RequestRefundGuide />;
  }

  if (category === "mall" && slug === "review") {
    return <PostProductReviewGuide />;
  }

  if (category === "mall" && slug === "payout-account") {
    return <OpenPayoutAccountGuide />;
  }

  if (category === "mall" && slug === "resale") {
    return <ListMarketGuide />;
  }

  if (category === "mall" && slug === "market") {
    return <ReviewResaleGuide />;
  }

  if (category === "mall" && slug === "trade") {
    return <TradeGuide />;
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