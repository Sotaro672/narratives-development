// frontend/mall/src/components/layout/header/useHeaderController.ts

import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { onAuthStateChanged, type User } from "firebase/auth";

import { fetchCart } from "../../../features/cart/api/cartApi";
import type { CartDTO, CartItemDTO } from "../../../features/shared/types/cart";
import { auth } from "../../../lib/firebase";
import type { HeaderActionState, HeaderProps } from "./types";

function getCartItemQty(item: CartItemDTO): number {
  if (!Number.isFinite(item.qty) || item.qty <= 0) {
    return 0;
  }

  return item.qty;
}

function sumCartItemQty(cart: CartDTO): number {
  return Object.values(cart.items).reduce(
    (sum, item) => sum + getCartItemQty(item),
    0,
  );
}

async function fetchCartItemCount(): Promise<number> {
  const cart = await fetchCart();
  return sumCartItemQty(cart);
}

export function useHeaderController({
  title,
  mode = "default",
  showEditButton = false,
  hideSettingsButton = false,
  hideAnnouncementButton = false,
  actionButtonLabel,
  onActionButtonClick,
  actionButtonDisabled = false,
  secondaryActionButtonLabel,
  onSecondaryActionButtonClick,
  secondaryActionButtonDisabled = false,
  tertiaryActionButtonLabel,
  onTertiaryActionButtonClick,
  tertiaryActionButtonDisabled = false,
  showCartButton = false,
  cartButtonLabel = "カート",
  onCartButtonClick,
  cartButtonDisabled = false,
  cartItemCount,
}: HeaderProps) {
  const navigate = useNavigate();

  const [settingsOpen, setSettingsOpen] = useState(false);
  const [currentUser, setCurrentUser] = useState<User | null>(null);
  const [authResolved, setAuthResolved] = useState(false);
  const [fetchedCartItemCount, setFetchedCartItemCount] = useState(0);

  useEffect(() => {
    const unsubscribe = onAuthStateChanged(auth, (user) => {
      setCurrentUser(user);
      setAuthResolved(true);
    });

    return unsubscribe;
  }, []);

  const isLoggedIn = !!currentUser;

  const hasActionButton =
    mode !== "signin" &&
    authResolved &&
    !!actionButtonLabel &&
    typeof onActionButtonClick === "function";

  const hasSecondaryActionButton =
    mode !== "signin" &&
    authResolved &&
    !!secondaryActionButtonLabel &&
    typeof onSecondaryActionButtonClick === "function";

  const hasTertiaryActionButton =
    mode !== "signin" &&
    authResolved &&
    !!tertiaryActionButtonLabel &&
    typeof onTertiaryActionButtonClick === "function";

  const shouldShowCartButton =
    mode !== "signin" &&
    authResolved &&
    !!showCartButton &&
    typeof onCartButtonClick === "function";

  useEffect(() => {
    let cancelled = false;

    async function loadCartItemCount() {
      if (
        !authResolved ||
        !currentUser ||
        !shouldShowCartButton ||
        typeof cartItemCount === "number"
      ) {
        setFetchedCartItemCount(0);
        return;
      }

      try {
        const count = await fetchCartItemCount();

        if (!cancelled) {
          setFetchedCartItemCount(count);
        }
      } catch {
        if (!cancelled) {
          setFetchedCartItemCount(0);
        }
      }
    }

    void loadCartItemCount();

    return () => {
      cancelled = true;
    };
  }, [
    authResolved,
    currentUser,
    shouldShowCartButton,
    cartItemCount,
  ]);

  const displayCartItemCount =
    typeof cartItemCount === "number"
      ? Math.max(0, cartItemCount)
      : fetchedCartItemCount;

  const displayTitle = title ?? "AMOL";

  const shouldShowLoginButton =
    mode !== "signin" &&
    authResolved &&
    !isLoggedIn;

  const shouldShowAnnouncementButton =
    mode !== "signin" &&
    authResolved &&
    !shouldShowLoginButton &&
    !hideAnnouncementButton;

  const shouldShowSettingsButton =
    mode !== "signin" &&
    authResolved &&
    isLoggedIn &&
    !showEditButton &&
    !hideSettingsButton &&
    !hasActionButton &&
    !hasSecondaryActionButton &&
    !hasTertiaryActionButton &&
    !shouldShowCartButton;

  const shouldShowEditButton =
    mode !== "signin" &&
    authResolved &&
    isLoggedIn &&
    showEditButton &&
    !hasActionButton &&
    !hasSecondaryActionButton &&
    !hasTertiaryActionButton &&
    !shouldShowCartButton;

  const closeSettings = () => {
    setSettingsOpen(false);
  };

  const toggleSettings = () => {
    setSettingsOpen((previous) => !previous);
  };

  const handleTitleClick = () => {
    navigate("/landing");
  };

  const actions: HeaderActionState = {
    hasActionButton,
    actionButtonLabel: actionButtonLabel ?? "",
    onActionButtonClick,
    actionButtonDisabled,
    hasSecondaryActionButton,
    secondaryActionButtonLabel: secondaryActionButtonLabel ?? "",
    onSecondaryActionButtonClick,
    secondaryActionButtonDisabled,
    hasTertiaryActionButton,
    tertiaryActionButtonLabel: tertiaryActionButtonLabel ?? "",
    onTertiaryActionButtonClick,
    tertiaryActionButtonDisabled,
    shouldShowCartButton,
    cartButtonLabel,
    onCartButtonClick,
    cartButtonDisabled,
    cartItemCount: displayCartItemCount,
    shouldShowLoginButton,
    shouldShowAnnouncementButton,
    shouldShowRoomCopyButton: false,
    shouldShowEditButton,
    shouldShowSettingsButton,
    copyButtonLabel: "",
    toggleSettings,
  };

  return {
    displayTitle,
    handleTitleClick,
    settingsOpen,
    shouldShowSettingsButton,
    closeSettings,
    actions,
  };
}