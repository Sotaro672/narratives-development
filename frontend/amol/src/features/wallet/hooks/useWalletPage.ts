// frontend/amol/src/features/wallet/hooks/useWalletPage.ts

import {
  useEffect,
  useState,
} from "react";
import {
  getAuth,
  onAuthStateChanged,
} from "firebase/auth";
import {
  useNavigate,
  useParams,
} from "react-router-dom";

import {
  getMyAvatar,
  getPublicAvatar,
} from "../../avatar/api/avatarApi";
import { LANDING_PATH } from "../../../lib/navigation";
import { getApiBaseUrl } from "../../../lib/apiBaseUrl";
import { getFirebaseIdToken } from "../../../lib/authToken";

import { fetchWalletOrders } from "../api/historyApi";
import { fetchMeWalletTokens } from "../api/walletTokenApi";

import type { WalletTabKey } from "../types";
import type { WalletOrder } from "../types/orderTypes";
import type {
  WalletDTO,
  WalletTokenItem,
} from "../types/tokenTypes";

export function useWalletPage() {
  const navigate = useNavigate();

  const {
    avatarId: routeAvatarId,
  } = useParams<{
    avatarId?: string;
  }>();

  const [
    avatarId,
    setAvatarId,
  ] = useState("");

  const [
    viewedAvatarId,
    setViewedAvatarId,
  ] = useState("");

  const [
    isOwnAvatar,
    setIsOwnAvatar,
  ] = useState(true);

  const [
    avatarName,
    setAvatarName,
  ] = useState("");

  const [
    avatarIcon,
    setAvatarIcon,
  ] = useState("");

  const [
    profile,
    setProfile,
  ] = useState("");

  const [
    wallet,
    setWallet,
  ] = useState<WalletDTO | null>(
    null,
  );

  const [
    walletTokens,
    setWalletTokens,
  ] = useState<WalletTokenItem[]>(
    [],
  );

  const [
    orderHistory,
    setOrderHistory,
  ] = useState<WalletOrder[]>(
    [],
  );

  const [
    orderLoading,
    setOrderLoading,
  ] = useState(false);

  const [
    orderError,
    setOrderError,
  ] = useState("");

  const [
    activeTab,
    setActiveTab,
  ] = useState<WalletTabKey>(
    "history",
  );

  const [
    loading,
    setLoading,
  ] = useState(true);

  const [
    error,
    setError,
  ] = useState("");

  const [
    tokenLoading,
    setTokenLoading,
  ] = useState(false);

  const [
    tokenError,
    setTokenError,
  ] = useState("");

  const [
    authResolved,
    setAuthResolved,
  ] = useState(false);

  useEffect(() => {
    const auth = getAuth();

    const unsubscribe =
      onAuthStateChanged(
        auth,
        (user) => {
          if (!user) {
            navigate(
              LANDING_PATH,
              {
                replace: true,
              },
            );

            return;
          }

          setAuthResolved(true);
        },
      );

    return () => unsubscribe();
  }, [navigate]);

  useEffect(() => {
    if (!authResolved) {
      return;
    }

    let isMounted = true;

    const fetchData =
      async () => {
        setLoading(true);
        setError("");

        setTokenLoading(true);
        setTokenError("");

        setOrderLoading(true);
        setOrderError("");

        try {
          const backendUrl =
            getApiBaseUrl();

          if (!backendUrl) {
            throw new Error(
              "VITE_API_BASE_URLが設定されていません。",
            );
          }

          const idToken =
            await getFirebaseIdToken();

          const meAvatar =
            await getMyAvatar({
              backendUrl,
              idToken,
            });

          if (!meAvatar) {
            throw new Error(
              "ログイン中のアバター情報が見つかりません。",
            );
          }

          const nextViewedAvatarId =
            routeAvatarId ||
            meAvatar.avatarId;

          const nextIsOwnAvatar =
            !routeAvatarId ||
            routeAvatarId ===
              meAvatar.avatarId;

          const viewedAvatar =
            nextIsOwnAvatar
              ? meAvatar
              : await getPublicAvatar({
                  backendUrl,
                  idToken,
                  avatarId:
                    nextViewedAvatarId,
                });

          if (!viewedAvatar) {
            throw new Error(
              "公開アバター情報が見つかりません。",
            );
          }

          if (!isMounted) {
            return;
          }

          setAvatarId(
            viewedAvatar.avatarId,
          );

          setViewedAvatarId(
            nextViewedAvatarId,
          );

          setIsOwnAvatar(
            nextIsOwnAvatar,
          );

          setAvatarName(
            viewedAvatar.avatarName,
          );

          setAvatarIcon(
            viewedAvatar.avatarIcon ??
              "",
          );

          setProfile(
            viewedAvatar.profile ??
              "",
          );

          if (!nextIsOwnAvatar) {
            setWallet(null);
            setWalletTokens([]);
            setOrderHistory([]);

            setTokenLoading(false);
            setOrderLoading(false);

            return;
          }

          try {
            const [
              tokenResult,
              orderResult,
            ] = await Promise.all([
              fetchMeWalletTokens(),

              fetchWalletOrders({
                backendUrl,
                idToken,
                page: 1,
                perPage: 20,
                sort: "createdAt",
                order: "desc",
              }),
            ]);

            if (!isMounted) {
              return;
            }

            setWallet(
              tokenResult.wallet,
            );

            setWalletTokens(
              tokenResult.tokens,
            );

            setOrderHistory(
              orderResult.items,
            );
          } catch (caught) {
            if (!isMounted) {
              return;
            }

            setWallet(null);
            setWalletTokens([]);
            setOrderHistory([]);

            const message =
              caught instanceof Error
                ? caught.message
                : "ウォレット情報の取得に失敗しました。";

            setTokenError(message);
            setOrderError(message);
          } finally {
            if (isMounted) {
              setTokenLoading(false);
              setOrderLoading(false);
            }
          }
        } catch (caught) {
          if (!isMounted) {
            return;
          }

          setError(
            caught instanceof Error
              ? caught.message
              : "ウォレット情報の取得に失敗しました。",
          );

          setWallet(null);
          setWalletTokens([]);
          setOrderHistory([]);

          setTokenLoading(false);
          setOrderLoading(false);
        } finally {
          if (isMounted) {
            setLoading(false);
          }
        }
      };

    void fetchData();

    return () => {
      isMounted = false;
    };
  }, [
    authResolved,
    navigate,
    routeAvatarId,
  ]);

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

    hasItems:
      orderHistory.length > 0,

    hasTokens:
      walletTokens.length > 0,

    pageTitle:
      avatarName ||
      "ウォレット",
  };
}