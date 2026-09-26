// frontend/mall/src/pages/CartPage.tsx

import { useNavigate } from "react-router-dom";

import { useMobilePortrait } from "../components/hooks/useMobilePortrait";
import Layout from "../components/layout/Layout";
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
      showFooter={isMobilePortrait}
      hideHamburgerMenu
      hideSettingsButton
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
            isPurchaseDisabled={isPurchaseDisabled}
            onRemoveItem={removeItem}
            onOpenItem={handleOpenItem}
            onPurchase={handlePurchase}
          />
        ) : null}
      </section>
    </Layout>
  );
}