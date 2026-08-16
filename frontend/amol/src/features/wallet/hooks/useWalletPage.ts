// frontend/amol/src/features/wallet/hooks/useWalletPage.ts

import { useEffect, useState } from "react";
import { getAuth, onAuthStateChanged } from "firebase/auth";
import { useNavigate, useParams } from "react-router-dom";

import { getMyAvatar, getPublicAvatar } from "../../avatar/api/avatarApi";
import { LANDING_PATH } from "../../../lib/navigation";
import { getApiBaseUrl } from "../../../lib/apiBaseUrl";
import { getFirebaseIdToken } from "../../../lib/authToken";
import { fetchWalletOrders } from "../api/historyApi";
import { fetchMeWalletTokens } from "../api/walletTokenApi";

import type { WalletTabKey } from "../types";
import type { WalletOrder } from "../../shared/types/orderTypes";
import type { WalletDTO, WalletTokenItem } from "../../shared/types/tokenTypes";

function getErrorMessage(caught: unknown, defaultMessage: string): string {
  return caught instanceof Error ? caught.message : defaultMessage;
}

export function useWalletPage() {
  const navigate = useNavigate();
  const { avatarId: routeAvatarId } = useParams<{ avatarId?: string }>();

  const [avatarId, setAvatarId] = useState("");
  const [viewedAvatarId, setViewedAvatarId] = useState("");
  const [isOwnAvatar, setIsOwnAvatar] = useState(true);
  const [avatarName, setAvatarName] = useState("");
  const [avatarIcon, setAvatarIcon] = useState("");
  const [profile, setProfile] = useState("");

  const [wallet, setWallet] = useState<WalletDTO | null>(null);
  const [walletTokens, setWalletTokens] = useState<WalletTokenItem[]>([]);
  const [orderHistory, setOrderHistory] = useState<WalletOrder[]>([]);

  const [orderLoading, setOrderLoading] = useState(false);
  const [orderError, setOrderError] = useState("");
  const [activeTab, setActiveTab] = useState<WalletTabKey>("history");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [tokenLoading, setTokenLoading] = useState(false);
  const [tokenError, setTokenError] = useState("");
  const [authResolved, setAuthResolved] = useState(false);

  useEffect(() => {
    const auth = getAuth();

    const unsubscribe = onAuthStateChanged(auth, (user) => {
      if (!user) {
        setAuthResolved(false);
        navigate(LANDING_PATH, { replace: true });
        return;
      }

      setAuthResolved(true);
    });

    return () => unsubscribe();
  }, [navigate]);

  useEffect(() => {
    if (!authResolved) return;

    let isMounted = true;

    const fetchData = async () => {
      setLoading(true);
      setError("");
      setTokenLoading(true);
      setTokenError("");
      setOrderLoading(true);
      setOrderError("");

      try {
        const meAvatar = await getMyAvatar();

        if (!meAvatar) {
          throw new Error("ログイン中のアバター情報が見つかりません。");
        }

        const nextViewedAvatarId = routeAvatarId || meAvatar.avatarId;
        const nextIsOwnAvatar = !routeAvatarId || routeAvatarId === meAvatar.avatarId;

        const viewedAvatar = nextIsOwnAvatar
          ? meAvatar
          : await getPublicAvatar({ avatarId: nextViewedAvatarId });

        if (!viewedAvatar) {
          throw new Error("公開アバター情報が見つかりません。");
        }

        if (!isMounted) return;

        setAvatarId(viewedAvatar.avatarId);
        setViewedAvatarId(viewedAvatar.avatarId);
        setIsOwnAvatar(nextIsOwnAvatar);
        setAvatarName(viewedAvatar.avatarName);
        setAvatarIcon(viewedAvatar.avatarIcon ?? "");
        setProfile(viewedAvatar.profile ?? "");

        if (!nextIsOwnAvatar) {
          setWallet(null);
          setWalletTokens([]);
          setOrderHistory([]);
          setTokenLoading(false);
          setOrderLoading(false);
          return;
        }

        const backendUrl = getApiBaseUrl();

        if (!backendUrl) {
          throw new Error("VITE_API_BASE_URLが設定されていません。");
        }

        const tokenPromise = fetchMeWalletTokens();
        const orderPromise = getFirebaseIdToken().then((idToken) =>
          fetchWalletOrders({
            backendUrl,
            idToken,
            page: 1,
            perPage: 20,
            sort: "createdAt",
            order: "desc",
          }),
        );

        const [tokenResult, orderResult] = await Promise.allSettled([
          tokenPromise,
          orderPromise,
        ]);

        if (!isMounted) return;

        if (tokenResult.status === "fulfilled") {
          setWallet(tokenResult.value.wallet);
          setWalletTokens(tokenResult.value.tokens);
          setTokenError("");
        } else {
          setWallet(null);
          setWalletTokens([]);
          setTokenError(
            getErrorMessage(
              tokenResult.reason,
              "ウォレット情報の取得に失敗しました。",
            ),
          );
        }

        if (orderResult.status === "fulfilled") {
          setOrderHistory(orderResult.value.items);
          setOrderError("");
        } else {
          setOrderHistory([]);
          setOrderError(
            getErrorMessage(
              orderResult.reason,
              "注文履歴の取得に失敗しました。",
            ),
          );
        }
      } catch (caught) {
        if (!isMounted) return;

        setError(
          getErrorMessage(
            caught,
            "ウォレット情報の取得に失敗しました。",
          ),
        );
        setWallet(null);
        setWalletTokens([]);
        setOrderHistory([]);
      } finally {
        if (isMounted) {
          setLoading(false);
          setTokenLoading(false);
          setOrderLoading(false);
        }
      }
    };

    void fetchData();

    return () => {
      isMounted = false;
    };
  }, [authResolved, routeAvatarId]);

  return {
    avatarId,
    viewedAvatarId,
    isOwnAvatar,
    avatarName,
    avatarIcon,
    profile,
    wallet,
    walletTokens,
    orderHistory,
    activeTab,
    setActiveTab,
    loading,
    error,
    tokenLoading,
    tokenError,
    orderLoading,
    orderError,
    hasItems: orderHistory.length > 0,
    hasTokens: walletTokens.length > 0,
    pageTitle: avatarName || "ウォレット",
  };
}