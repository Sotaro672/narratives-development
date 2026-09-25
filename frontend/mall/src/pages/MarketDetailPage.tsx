// frontend/mall/src/pages/MarketDetailPage.tsx

import { useLocation, useNavigate, useParams } from "react-router-dom";

import { useMobilePortrait } from "../components/hooks/useMobilePortrait";
import Layout from "../components/layout/Layout";

import { addResaleCartItem } from "../features/cart/api/cartApi";
import MarketDetailContent from "../features/market/presentation/components/MarketDetailContent";
import { useMarketDetailPage } from "../features/market/presentation/hooks/useMarketDetailPage";
import ReportModal from "../features/report/components/ReportModal";
import { useReport } from "../features/report/hooks/useReport";

import "../styles/page-layout.css";
import "../styles/market-detail-page.css";

type MarketDetailRouteState = {
  reviewContext?: boolean;
};

export default function MarketDetailPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const { resaleId } = useParams<{ resaleId: string }>();
  const isMobilePortrait = useMobilePortrait();

  const routeState = location.state as MarketDetailRouteState | null;
  const reviewContext = routeState?.reviewContext === true;

  const detail = useMarketDetailPage({
    resaleId,
    addResaleProductToCart: addResaleCartItem,
    reviewContext,
  });

  const {
    sellerAvatarId,
    handleAddToCart,
  } = detail;

  const {
    target: reportTarget,
    isOpen: reportOpen,
    reason: reportReason,
    detail: reportDetail,
    submitting: reportSubmitting,
    error: reportError,
    result: reportResult,
    canSubmit: canSubmitReport,
    openResaleReport,
    close: closeReport,
    setReason: setReportReason,
    setDetail: setReportDetail,
    submit: submitReport,
  } = useReport();

  const normalizedResaleId = resaleId?.trim() ?? "";

  async function handleAddToCartAndOpenCart(): Promise<void> {
    const added = await handleAddToCart();
    if (!added) return;
    navigate("/cart");
  }

  function handleOpenSellerAvatar() {
    if (!sellerAvatarId) return;
    navigate(`/avatars/${encodeURIComponent(sellerAvatarId)}`);
  }

  function handleOpenResaleChat() {
    if (!normalizedResaleId) return;

    navigate(`/chats/resales/${encodeURIComponent(normalizedResaleId)}`, {
      state: {
        source: "market",
      },
    });
  }

  function handleOpenResaleReport() {
    if (!normalizedResaleId || reportSubmitting) return;

    openResaleReport({
      resaleId: normalizedResaleId,
    });
  }

  return (
    <Layout
      title="AMOL"
      titleClickable
      mode="mypage"
      showHeader={!isMobilePortrait}
      hideAnnouncementButton
      hideSettingsButton
      hideHamburgerMenu
      showCartButton
      cartButtonLabel="カート"
      onCartButtonClick={() => navigate("/cart")}
    >
      <MarketDetailContent
        detail={detail}
        onOpenSeller={handleOpenSellerAvatar}
        onOpenResaleChat={handleOpenResaleChat}
        onOpenResaleReport={handleOpenResaleReport}
        onAddToCart={handleAddToCartAndOpenCart}
        reportSubmitting={reportSubmitting}
      />

      <ReportModal
        open={reportOpen}
        targetType={reportTarget?.type}
        reason={reportReason}
        detail={reportDetail}
        submitting={reportSubmitting}
        error={reportError}
        result={reportResult}
        canSubmit={canSubmitReport}
        onReasonChange={setReportReason}
        onDetailChange={setReportDetail}
        onSubmit={submitReport}
        onClose={closeReport}
      />
    </Layout>
  );
}