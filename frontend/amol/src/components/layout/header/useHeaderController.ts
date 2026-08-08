// frontend/amol/src/components/layout/header/useHeaderController.ts

import {
  useEffect,
  useState,
} from "react";
import {
  useLocation,
  useNavigate,
} from "react-router-dom";
import {
  onAuthStateChanged,
  type User,
} from "firebase/auth";

import { fetchCart } from "../../../features/cart/api/cartApi";
import type {
  CartDTO,
  CartItemDTO,
} from "../../../features/shared/types/cart";
import { auth } from "../../../lib/firebase";
import { WALLET_PATH } from "../../../lib/navigation";
import type {
  HeaderActionState,
  HeaderProps,
} from "./types";

function getCartItemQty(
  item: CartItemDTO,
): number {
  if (
    !Number.isFinite(item.qty) ||
    item.qty <= 0
  ) {
    return 0;
  }

  return item.qty;
}

function sumCartItemQty(
  cart: CartDTO,
): number {
  return Object.values(cart.items).reduce(
    (sum, item) =>
      sum + getCartItemQty(item),
    0,
  );
}

async function fetchCartItemCount(): Promise<number> {
  const cart = await fetchCart();

  return sumCartItemQty(cart);
}

export function useHeaderController({
  title,
  showBackButton = false,
  backTo = WALLET_PATH,
  mode = "default",
  showEditButton = false,
  hideHamburgerMenu = false,
  hideSettingsButton = false,
  hideAnnouncementButton = false,

  onBackButtonClick,

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
  const location = useLocation();
  const navigate = useNavigate();

  const [
    menuOpen,
    setMenuOpen,
  ] = useState(false);

  const [
    settingsOpen,
    setSettingsOpen,
  ] = useState(false);

  const [
    currentUser,
    setCurrentUser,
  ] = useState<User | null>(null);

  const [
    authResolved,
    setAuthResolved,
  ] = useState(false);

  const [
    isDesktop,
    setIsDesktop,
  ] = useState(false);

  const [
    fetchedCartItemCount,
    setFetchedCartItemCount,
  ] = useState(0);

  useEffect(() => {
    setMenuOpen(false);
    setSettingsOpen(false);
  }, [
    location.pathname,
  ]);

  useEffect(() => {
    const unsubscribe =
      onAuthStateChanged(
        auth,
        (user) => {
          setCurrentUser(user);
          setAuthResolved(true);
        },
      );

    return unsubscribe;
  }, []);

  useEffect(() => {
    if (
      typeof window === "undefined"
    ) {
      return;
    }

    const desktopQuery =
      window.matchMedia(
        "(min-width: 1024px)",
      );

    const updateViewportState =
      () => {
        setIsDesktop(
          desktopQuery.matches,
        );
      };

    updateViewportState();

    if (
      typeof desktopQuery.addEventListener ===
      "function"
    ) {
      desktopQuery.addEventListener(
        "change",
        updateViewportState,
      );

      return () => {
        desktopQuery.removeEventListener(
          "change",
          updateViewportState,
        );
      };
    }

    desktopQuery.addListener(
      updateViewportState,
    );

    return () => {
      desktopQuery.removeListener(
        updateViewportState,
      );
    };
  }, []);

  const isLoggedIn =
    !!currentUser;

  const isContactPage =
    location.pathname === "/contact";

  const isInfoPage =
    location.pathname === "/" ||
    location.pathname ===
      "/landing" ||
    location.pathname ===
      "/specified-commercial-transactions" ||
    location.pathname ===
      "/terms" ||
    location.pathname ===
      "/privacy-policy" ||
    location.pathname ===
      "/contact";

  const isRoomDetailPage =
    /^\/lists\/[^/]+$/.test(
      location.pathname,
    );

  const shouldHideHamburgerMenu =
    hideHamburgerMenu ||
    isRoomDetailPage;

  const hasActionButton =
    mode !== "signin" &&
    authResolved &&
    !!actionButtonLabel &&
    typeof onActionButtonClick ===
      "function";

  const hasSecondaryActionButton =
    mode !== "signin" &&
    authResolved &&
    !!secondaryActionButtonLabel &&
    typeof onSecondaryActionButtonClick ===
      "function";

  const hasTertiaryActionButton =
    mode !== "signin" &&
    authResolved &&
    !!tertiaryActionButtonLabel &&
    typeof onTertiaryActionButtonClick ===
      "function";

  const shouldShowCartButton =
    mode !== "signin" &&
    authResolved &&
    !!showCartButton &&
    typeof onCartButtonClick ===
      "function";

  useEffect(() => {
    let cancelled = false;

    async function loadCartItemCount() {
      if (
        !authResolved ||
        !currentUser ||
        !shouldShowCartButton ||
        typeof cartItemCount ===
          "number"
      ) {
        setFetchedCartItemCount(0);
        return;
      }

      try {
        const count =
          await fetchCartItemCount();

        if (!cancelled) {
          setFetchedCartItemCount(
            count,
          );
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
    location.pathname,
  ]);

  const displayCartItemCount =
    typeof cartItemCount === "number"
      ? Math.max(0, cartItemCount)
      : fetchedCartItemCount;

  const displayTitle =
    title ?? "AMOL";

  const shouldShowBackButton =
    isContactPage
      ? isLoggedIn
      : isInfoPage
        ? false
        : showBackButton;

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

  const shouldShowGuestMenuButton =
    mode !== "signin" &&
    authResolved &&
    !isLoggedIn &&
    !isDesktop &&
    !shouldHideHamburgerMenu;

  const shouldShowAuthenticatedMenuButton =
    mode !== "signin" &&
    authResolved &&
    isLoggedIn &&
    !shouldHideHamburgerMenu;

  const shouldShowLandscapeSidebarMenuButton =
    false;

  const shouldShowMenuButton =
    shouldShowGuestMenuButton ||
    shouldShowAuthenticatedMenuButton;

  const closeMenu = () => {
    setMenuOpen(false);
  };

  const closeSettings = () => {
    setSettingsOpen(false);
  };

  const toggleMenu = () => {
    setSettingsOpen(false);

    setMenuOpen(
      (previous) => !previous,
    );
  };

  const toggleSettings = () => {
    setMenuOpen(false);

    setSettingsOpen(
      (previous) => !previous,
    );
  };

  const handleBack = () => {
    if (onBackButtonClick) {
      void onBackButtonClick();
      return;
    }

    const normalizedBackTo =
      backTo.trim();

    navigate(
      normalizedBackTo ||
        WALLET_PATH,
    );
  };

  const handleTitleClick = () => {
    navigate("/landing");
  };

  const actions: HeaderActionState = {
    hasActionButton,
    actionButtonLabel:
      actionButtonLabel ?? "",
    onActionButtonClick,
    actionButtonDisabled,

    hasSecondaryActionButton,
    secondaryActionButtonLabel:
      secondaryActionButtonLabel ??
      "",
    onSecondaryActionButtonClick,
    secondaryActionButtonDisabled,

    hasTertiaryActionButton,
    tertiaryActionButtonLabel:
      tertiaryActionButtonLabel ??
      "",
    onTertiaryActionButtonClick,
    tertiaryActionButtonDisabled,

    shouldShowCartButton,
    cartButtonLabel,
    onCartButtonClick,
    cartButtonDisabled,
    cartItemCount:
      displayCartItemCount,

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
    menuOpen,
    settingsOpen,
    shouldShowMenuButton,
    shouldShowBackButton,
    shouldShowLandscapeSidebarMenuButton,
    shouldShowSettingsButton,
    closeMenu,
    closeSettings,
    handleBack,
    toggleMenu,
    actions,
  };
}