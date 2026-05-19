// frontend/amol/src/components/layout/header/useHeaderController.ts
import { useEffect, useMemo, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { onAuthStateChanged, type User } from "firebase/auth";

import { auth } from "../../../lib/firebase";
import { WALLET_PATH } from "../../../lib/navigation";
import type { HeaderActionState, HeaderProps } from "./types";

type MeAvatarStateResponse = {
  avatarId?: string;
};

type CartItemDTO = {
  qty?: number;
  quantity?: number;
  [key: string]: unknown;
};

type CartDTO = {
  avatarId?: string;
  items?: Record<string, CartItemDTO> | CartItemDTO[];
};

function getApiBaseUrl(): string {
  const env = import.meta.env.VITE_API_BASE_URL;

  if (typeof env === "string" && env.trim() !== "") {
    return env.replace(/\/$/, "");
  }

  return "";
}

function getCartItemQty(item: CartItemDTO): number {
  const rawQty = item.qty ?? item.quantity;

  if (typeof rawQty === "number" && Number.isFinite(rawQty)) {
    return Math.max(0, rawQty);
  }

  if (typeof rawQty === "string") {
    const parsed = Number(rawQty);

    if (Number.isFinite(parsed)) {
      return Math.max(0, parsed);
    }
  }

  return 0;
}

function sumCartItemQty(cart: CartDTO): number {
  const items = cart.items;

  if (!items) {
    return 0;
  }

  if (Array.isArray(items)) {
    return items.reduce((sum, item) => sum + getCartItemQty(item), 0);
  }

  return Object.values(items).reduce(
    (sum, item) => sum + getCartItemQty(item),
    0
  );
}

async function readResponseErrorMessage(response: Response): Promise<string> {
  const contentType = response.headers.get("content-type") ?? "";

  if (contentType.includes("application/json")) {
    const data = (await response.json().catch(() => null)) as
      | { error?: unknown; message?: unknown }
      | null;

    if (typeof data?.error === "string" && data.error.trim() !== "") {
      return data.error;
    }

    if (typeof data?.message === "string" && data.message.trim() !== "") {
      return data.message;
    }
  }

  const text = await response.text().catch(() => "");

  if (text.trim() !== "") {
    return text;
  }

  return "リクエストに失敗しました。";
}

async function fetchCurrentAvatarId(args: {
  apiBaseUrl: string;
  currentUser: User;
}): Promise<string> {
  const { apiBaseUrl, currentUser } = args;
  const idToken = await currentUser.getIdToken();

  const response = await fetch(`${apiBaseUrl}/mall/me/avatars/state`, {
    method: "GET",
    headers: {
      Accept: "application/json",
      Authorization: `Bearer ${idToken}`,
    },
    credentials: "include",
  });

  if (!response.ok) {
    const message = await readResponseErrorMessage(response);
    throw new Error(message || "現在のアバター情報の取得に失敗しました。");
  }

  const contentType = response.headers.get("content-type") ?? "";

  if (!contentType.includes("application/json")) {
    throw new Error("現在のアバター情報APIがJSON以外を返しました。");
  }

  const data = (await response.json()) as MeAvatarStateResponse;
  const avatarId = data.avatarId?.trim();

  if (!avatarId) {
    throw new Error("現在のavatarIdが見つかりません。");
  }

  return avatarId;
}

async function fetchCartItemCount(args: {
  apiBaseUrl: string;
  currentUser: User;
}): Promise<number> {
  const { apiBaseUrl, currentUser } = args;

  const avatarId = await fetchCurrentAvatarId({
    apiBaseUrl,
    currentUser,
  });

  const idToken = await currentUser.getIdToken();
  const searchParams = new URLSearchParams({
    avatarId,
  });

  let response = await fetch(
    `${apiBaseUrl}/mall/me/cart/query?${searchParams.toString()}`,
    {
      method: "GET",
      headers: {
        Accept: "application/json",
        Authorization: `Bearer ${idToken}`,
      },
      credentials: "include",
    }
  );

  if (response.status === 404) {
    response = await fetch(
      `${apiBaseUrl}/mall/me/cart?${searchParams.toString()}`,
      {
        method: "GET",
        headers: {
          Accept: "application/json",
          Authorization: `Bearer ${idToken}`,
        },
        credentials: "include",
      }
    );
  }

  if (!response.ok) {
    return 0;
  }

  const contentType = response.headers.get("content-type") ?? "";

  if (!contentType.includes("application/json")) {
    return 0;
  }

  const data = (await response.json().catch(() => null)) as CartDTO | null;

  if (!data) {
    return 0;
  }

  return sumCartItemQty(data);
}

export function useHeaderController({
  title,
  showBackButton = false,
  backTo = WALLET_PATH,
  mode = "default",
  showEditButton = false,
  hideHamburgerMenu = false,
  hideSettingsButton = false,
  onBackButtonClick,
  actionButtonLabel,
  onActionButtonClick,
  actionButtonDisabled = false,
  secondaryActionButtonLabel,
  onSecondaryActionButtonClick,
  secondaryActionButtonDisabled = false,
  showCartButton = false,
  cartButtonLabel = "カート",
  onCartButtonClick,
  cartButtonDisabled = false,
  cartItemCount,
}: HeaderProps) {
  const location = useLocation();
  const navigate = useNavigate();

  const [menuOpen, setMenuOpen] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [currentUser, setCurrentUser] = useState<User | null>(null);
  const [authResolved, setAuthResolved] = useState(false);
  const [isLandscape, setIsLandscape] = useState(false);
  const [fetchedCartItemCount, setFetchedCartItemCount] = useState(0);

  const apiBaseUrl = useMemo(() => getApiBaseUrl(), []);

  useEffect(() => {
    setMenuOpen(false);
    setSettingsOpen(false);
  }, [location.pathname]);

  useEffect(() => {
    const unsubscribe = onAuthStateChanged(auth, (user) => {
      setCurrentUser(user);
      setAuthResolved(true);
    });

    return unsubscribe;
  }, []);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    const landscapeQuery = window.matchMedia("(orientation: landscape)");

    const updateViewportState = () => {
      setIsLandscape(landscapeQuery.matches);
    };

    updateViewportState();

    if (typeof landscapeQuery.addEventListener === "function") {
      landscapeQuery.addEventListener("change", updateViewportState);

      return () => {
        landscapeQuery.removeEventListener("change", updateViewportState);
      };
    }

    landscapeQuery.addListener(updateViewportState);

    return () => {
      landscapeQuery.removeListener(updateViewportState);
    };
  }, []);

  const isLoggedIn = !!currentUser;
  const isContactPage = location.pathname === "/contact";

  const isInfoPage =
    location.pathname === "/" ||
    location.pathname === "/specified-commercial-transactions" ||
    location.pathname === "/terms" ||
    location.pathname === "/privacy-policy" ||
    location.pathname === "/contact";

  const isRoomDetailPage = /^\/lists\/[^/]+$/.test(location.pathname);

  const shouldHideHamburgerMenu = hideHamburgerMenu || isRoomDetailPage;

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
        const count = await fetchCartItemCount({
          apiBaseUrl,
          currentUser,
        });

        if (!cancelled) {
          setFetchedCartItemCount(count);
        }
      } catch {
        if (!cancelled) {
          setFetchedCartItemCount(0);
        }
      }
    }

    loadCartItemCount();

    return () => {
      cancelled = true;
    };
  }, [
    apiBaseUrl,
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

  const displayTitle = title ?? "AMOL";

  const shouldShowBackButton = isContactPage
    ? isLoggedIn
    : isInfoPage
      ? false
      : showBackButton;

  const shouldShowLoginButton =
    mode !== "signin" && authResolved && !isLoggedIn;

  const shouldShowSettingsButton =
    mode !== "signin" &&
    authResolved &&
    isLoggedIn &&
    !showEditButton &&
    !hideSettingsButton &&
    !hasActionButton &&
    !hasSecondaryActionButton &&
    !shouldShowCartButton;

  const shouldShowEditButton =
    mode !== "signin" &&
    authResolved &&
    isLoggedIn &&
    showEditButton &&
    !hasActionButton &&
    !hasSecondaryActionButton &&
    !shouldShowCartButton;

  const shouldShowGuestMenuButton =
    mode !== "signin" &&
    authResolved &&
    !isLoggedIn &&
    !shouldHideHamburgerMenu;

  const shouldShowLandscapeSidebarMenuButton =
    mode !== "signin" &&
    authResolved &&
    isLoggedIn &&
    isLandscape &&
    !shouldHideHamburgerMenu &&
    !hasActionButton &&
    !hasSecondaryActionButton;

  const shouldShowMenuButton =
    shouldShowGuestMenuButton || shouldShowLandscapeSidebarMenuButton;

  const closeMenu = () => {
    setMenuOpen(false);
  };

  const closeSettings = () => {
    setSettingsOpen(false);
  };

  const toggleMenu = () => {
    setSettingsOpen(false);
    setMenuOpen((prev) => !prev);
  };

  const toggleSettings = () => {
    setMenuOpen(false);
    setSettingsOpen((prev) => !prev);
  };

  const handleBack = () => {
    if (onBackButtonClick) {
      void onBackButtonClick();
      return;
    }

    const normalizedBackTo = backTo.trim();

    navigate(normalizedBackTo || WALLET_PATH);
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

    shouldShowCartButton,
    cartButtonLabel,
    onCartButtonClick,
    cartButtonDisabled,
    cartItemCount: displayCartItemCount,

    shouldShowLoginButton,
    shouldShowRoomCopyButton: false,
    shouldShowEditButton,
    shouldShowSettingsButton,
    copyButtonLabel: "",

    toggleSettings,
  };

  return {
    displayTitle,
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