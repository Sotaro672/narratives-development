// frontend/amol/src/pages/CartPage.tsx

import { useNavigate } from "react-router-dom";
import Layout from "../components/layout/Layout";
import { useMobilePortrait } from "../components/hooks/useMobilePortrait";
import CartContent from "../features/cart/presentation/components/CartContent";
import CartPageEmpty from "../features/cart/presentation/components/CartPageEmpty";
import CartPageError from "../features/cart/presentation/components/CartPageError";
import CartPageLoading from "../features/cart/presentation/components/CartPageLoading";
import { useCartPage } from "../features/cart/presentation/hooks/useCartPage";
import "../styles/cart-page.css";

export default function CartPage() {
  const navigate = useNavigate();
  const isMobilePortrait = useMobilePortrait();

  const {
    items,
    totalAmount,
    loading,
    error,
    removingItemKey,
    isPurchaseDisabled,
    removeItem,
    reload,
  } = useCartPage();

  const hasItems = items.length > 0;

  function handlePurchase() {
    if (isPurchaseDisabled) {
      return;
    }

    navigate("/payments/cart");
  }

  function handleOpenItem(path: string) {
    const normalizedPath = path.trim();

    if (!normalizedPath) {
      return;
    }

    navigate(normalizedPath);
  }

  return (
    <Layout
      title="カート"
      titleClickable={false}
      mode="mypage"
      showBackButton
      backTo="/lists"
      showFooter={isMobilePortrait}
      hideHamburgerMenu
      hideSettingsButton
      actionButtonLabel={isMobilePortrait ? undefined : "購入する"}
      onActionButtonClick={isMobilePortrait ? undefined : handlePurchase}
      actionButtonDisabled={isPurchaseDisabled}
      footerProps={
        isMobilePortrait
          ? {
              variant: "action",
              buttonLabel: "購入する",
              disabled: isPurchaseDisabled,
              onButtonClick: handlePurchase,
            }
          : undefined
      }
    >
      <section className="content-page-section cart-page-section-root">
        {loading ? <CartPageLoading /> : null}

        {!loading && error ? (
          <CartPageError
            error={error}
            onRetry={() => {
              void reload();
            }}
          />
        ) : null}

        {!loading && !error && !hasItems ? <CartPageEmpty /> : null}

        {!loading && !error && hasItems ? (
          <CartContent
            items={items}
            totalAmount={totalAmount}
            removingItemKey={removingItemKey}
            onRemoveItem={removeItem}
            onOpenItem={handleOpenItem}
          />
        ) : null}
      </section>
    </Layout>
  );
}