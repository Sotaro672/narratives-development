// frontend/amol/src/pages/MarketDetailPage.tsx

import { useNavigate, useParams } from "react-router-dom";

import Layout from "../components/layout/Layout";

import { addResaleCartItem } from "../features/cart/api/cartApi";
import MarketDetailContent from "../features/market/presentation/components/MarketDetailContent";
import { useMarketDetailPage } from "../features/market/presentation/hooks/useMarketDetailPage";

import "../styles/page-layout.css";
import "../styles/market-detail-page.css";

export default function MarketDetailPage() {
  const navigate = useNavigate();
  const { resaleId } = useParams<{ resaleId: string }>();

  const detail = useMarketDetailPage({
    resaleId,
    addResaleProductToCart: addResaleCartItem,
  });

  const {
    title,
    addingToCart,
    canAddToCart,
    sellerAvatarId,
    handleAddToCart,
  } = detail;

  const addToCartButtonLabel = addingToCart ? "追加中" : "カートに入れる";

  async function handleAddToCartAndOpenCart(): Promise<void> {
    const added = await handleAddToCart();

    if (!added) {
      return;
    }

    navigate("/cart");
  }

  function handleOpenSellerAvatar() {
    if (!sellerAvatarId) {
      return;
    }

    navigate(`/avatars/${encodeURIComponent(sellerAvatarId)}`);
  }

  return (
    <Layout
      title={title}
      titleClickable={false}
      showBackButton
      onBackButtonClick={() => navigate(-1)}
      hideAnnouncementButton
      hideSettingsButton
      hideHamburgerMenu
      showCartButton
      cartButtonLabel="カート"
      onCartButtonClick={() => navigate("/cart")}
      actionButtonLabel={addToCartButtonLabel}
      onActionButtonClick={handleAddToCartAndOpenCart}
      actionButtonDisabled={!canAddToCart}
      showFooter
      footerProps={{
        variant: "action",
        buttonLabel: addToCartButtonLabel,
        disabled: !canAddToCart,
        onButtonClick: handleAddToCartAndOpenCart,
      }}
    >
      <MarketDetailContent
        detail={detail}
        onOpenSeller={handleOpenSellerAvatar}
      />
    </Layout>
  );
}