// frontend/amol/src/pages/MarketDetailPage.tsx

import { useNavigate, useParams } from "react-router-dom";

import Layout from "../components/layout/Layout";
import Button from "../components/ui/Button";

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
    item,
    title,
    isLiked,
    loading,
    loadingLike,
    addingToCart,
    updatingLike,
    canAddToCart,
    error,
    likeErrorMessage,
    sellerAvatarId,
    handleToggleLike,
    handleAddToCart,
  } = detail;

  const addToCartButtonLabel = addingToCart
    ? "追加中"
    : "カートに入れる";

  const likeButtonLabel = loadingLike
    ? "お気に入り確認中"
    : updatingLike
      ? isLiked
        ? "お気に入り解除中"
        : "お気に入り追加中"
      : isLiked
        ? "お気に入りから解除"
        : "お気に入りに追加";

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

  function handleOpenResaleChat() {
    const normalizedResaleId = resaleId?.trim() ?? "";

    if (!normalizedResaleId) {
      return;
    }

    navigate(`/chats/resales/${encodeURIComponent(normalizedResaleId)}`, {
      state: {
        source: "market",
      },
    });
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
      {!loading && !error && item ? (
        <div className="market-detail-page">
          <Button
            variant="secondary"
            fullWidth
            disabled={loadingLike || updatingLike}
            aria-pressed={isLiked}
            onClick={handleToggleLike}
          >
            {likeButtonLabel}
          </Button>

          {likeErrorMessage ? (
            <p className="market-detail-page__cart-error" role="alert">
              {likeErrorMessage}
            </p>
          ) : null}
        </div>
      ) : null}

      <MarketDetailContent
        detail={detail}
        onOpenSeller={handleOpenSellerAvatar}
        onOpenResaleChat={handleOpenResaleChat}
      />
    </Layout>
  );
}