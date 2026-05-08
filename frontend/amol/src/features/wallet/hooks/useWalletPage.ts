// frontend/amol/src/features/wallet/hooks/useWalletPage.ts
import { useCallback, useEffect, useState } from "react";
import { getAuth, onAuthStateChanged } from "firebase/auth";
import { useNavigate, useParams } from "react-router-dom";

import { LANDING_PATH } from "../../../lib/navigation";
import {
  fetchPublicWalletAvatar,
  fetchWalletAvatar,
} from "../api/avatarApi";
import { fetchWalletOrders } from "../api/historyApi";
import {
  fetchPublicWalletFollowState,
  followAvatar,
} from "../api/walletFollowApi";
import { fetchMeWalletTokens } from "../api/walletTokenApi";
import type { WalletTabKey } from "../types";
import type {
  PublicWalletFollowTabKey,
  PublicWalletFollowUser,
} from "../types/followTypes";
import type { WalletOrder } from "../types/orderTypes";
import type { WalletDTO, WalletTokenItem } from "../types/tokenTypes";

const BACKEND_BASE_URL = import.meta.env.VITE_API_BASE_URL;

export function useWalletPage() {
  const navigate = useNavigate();
  const { avatarId: routeAvatarId } = useParams<{ avatarId?: string }>();

  const [avatarId, setAvatarId] = useState("");
  const [viewedAvatarId, setViewedAvatarId] = useState("");
  const [isOwnAvatar, setIsOwnAvatar] = useState(true);

  const [avatarName, setAvatarName] = useState("");
  const [avatarIcon, setAvatarIcon] = useState("");
  const [profile, setProfile] = useState("");
  const [followerCount, setFollowerCount] = useState(0);
  const [followingCount, setFollowingCount] = useState(0);

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

  const [isFollowing, setIsFollowing] = useState(false);
  const [followPosting, setFollowPosting] = useState(false);
  const [followError, setFollowError] = useState("");

  const [publicFollowActiveTab, setPublicFollowActiveTab] =
    useState<PublicWalletFollowTabKey>("following");
  const [publicFollowing, setPublicFollowing] = useState<
    PublicWalletFollowUser[]
  >([]);
  const [publicFollowers, setPublicFollowers] = useState<
    PublicWalletFollowUser[]
  >([]);
  const [publicFollowLoading, setPublicFollowLoading] = useState(false);
  const [publicFollowError, setPublicFollowError] = useState("");

  useEffect(() => {
    const auth = getAuth();

    const unsubscribe = onAuthStateChanged(auth, (user) => {
      if (!user) {
        navigate(LANDING_PATH, { replace: true });
        return;
      }

      setAuthResolved(true);
    });

    return () => unsubscribe();
  }, [navigate]);

  useEffect(() => {
    if (!authResolved) {
      return;
    }

    let isMounted = true;

    const fetchData = async () => {
      setLoading(true);
      setError("");
      setTokenLoading(true);
      setTokenError("");
      setOrderLoading(true);
      setOrderError("");
      setFollowError("");
      setIsFollowing(false);

      setPublicFollowActiveTab("following");
      setPublicFollowing([]);
      setPublicFollowers([]);
      setPublicFollowLoading(false);
      setPublicFollowError("");

      try {
        if (!BACKEND_BASE_URL) {
          throw new Error("VITE_API_BASE_URL is not configured.");
        }

        const auth = getAuth();
        const user = auth.currentUser;

        if (!user) {
          navigate(LANDING_PATH, { replace: true });
          return;
        }

        const idToken = await user.getIdToken();

        const meAvatar = await fetchWalletAvatar({
          backendUrl: BACKEND_BASE_URL,
          idToken,
        });

        const nextViewedAvatarId = routeAvatarId || meAvatar.avatarId;
        const nextIsOwnAvatar =
          !routeAvatarId || routeAvatarId === meAvatar.avatarId;

        const viewedAvatar = nextIsOwnAvatar
          ? meAvatar
          : await fetchPublicWalletAvatar({
              backendUrl: BACKEND_BASE_URL,
              idToken,
              avatarId: nextViewedAvatarId,
            });

        if (!isMounted) return;

        setAvatarId(viewedAvatar.avatarId);
        setViewedAvatarId(nextViewedAvatarId);
        setIsOwnAvatar(nextIsOwnAvatar);

        setAvatarName(viewedAvatar.avatarName);
        setAvatarIcon(viewedAvatar.avatarIcon);
        setProfile(viewedAvatar.profile);
        setFollowerCount(viewedAvatar.followerCount);
        setFollowingCount(viewedAvatar.followingCount);

        if (!nextIsOwnAvatar) {
          setWallet(null);
          setWalletTokens([]);
          setOrderHistory([]);
          setTokenLoading(false);
          setOrderLoading(false);
          setPublicFollowLoading(true);
          setPublicFollowError("");

          try {
            const followState = await fetchPublicWalletFollowState({
              backendUrl: BACKEND_BASE_URL,
              idToken,
              avatarId: nextViewedAvatarId,
            });

            if (!isMounted) return;

            setFollowerCount(followState.followerCount);
            setFollowingCount(followState.followingCount);
            setPublicFollowing(followState.following);
            setPublicFollowers(followState.followers);
          } catch (err) {
            if (!isMounted) return;

            setPublicFollowing([]);
            setPublicFollowers([]);
            setPublicFollowError(
              err instanceof Error
                ? err.message
                : "フォロー情報の取得に失敗しました。"
            );
          } finally {
            if (isMounted) {
              setPublicFollowLoading(false);
            }
          }

          return;
        }

        try {
          const [tokenResult, orderResult] = await Promise.all([
            fetchMeWalletTokens({
              backendUrl: BACKEND_BASE_URL,
              idToken,
            }),
            fetchWalletOrders({
              backendUrl: BACKEND_BASE_URL,
              idToken,
              page: 1,
              perPage: 20,
              sort: "createdAt",
              order: "desc",
            }),
          ]);

          if (!isMounted) return;

          setWallet(tokenResult.wallet);
          setWalletTokens(tokenResult.tokens);
          setOrderHistory(orderResult.items);
        } catch (err) {
          if (!isMounted) return;

          setWallet(null);
          setWalletTokens([]);
          setOrderHistory([]);

          const message =
            err instanceof Error
              ? err.message
              : "ウォレット情報の取得に失敗しました。";

          setTokenError(message);
          setOrderError(message);
        } finally {
          if (isMounted) {
            setTokenLoading(false);
            setOrderLoading(false);
          }
        }
      } catch (err) {
        if (!isMounted) return;

        setError(
          err instanceof Error
            ? err.message
            : "ウォレット情報の取得に失敗しました。"
        );
        setWallet(null);
        setWalletTokens([]);
        setOrderHistory([]);
        setPublicFollowing([]);
        setPublicFollowers([]);
        setTokenLoading(false);
        setOrderLoading(false);
        setPublicFollowLoading(false);
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
  }, [authResolved, navigate, routeAvatarId]);

  const handleFollowAvatar = useCallback(async () => {
    if (isOwnAvatar || !avatarId || followPosting || isFollowing) {
      return;
    }

    setFollowPosting(true);
    setFollowError("");

    try {
      if (!BACKEND_BASE_URL) {
        throw new Error("VITE_API_BASE_URL is not configured.");
      }

      const auth = getAuth();
      const user = auth.currentUser;

      if (!user) {
        navigate(LANDING_PATH, { replace: true });
        return;
      }

      const idToken = await user.getIdToken();

      const nextState = await followAvatar({
        backendUrl: BACKEND_BASE_URL,
        idToken,
        targetAvatarId: avatarId,
      });

      setIsFollowing(true);
      setFollowerCount(nextState.followerCount ?? followerCount + 1);
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "フォローに失敗しました。";

      setFollowError(message);
      throw err;
    } finally {
      setFollowPosting(false);
    }
  }, [avatarId, followPosting, followerCount, isFollowing, isOwnAvatar, navigate]);

  return {
    avatarId,
    viewedAvatarId,
    isOwnAvatar,
    avatarName,
    avatarIcon,
    profile,
    followerCount,
    followingCount,
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
    isFollowing,
    followPosting,
    followError,
    handleFollowAvatar,
    publicFollowActiveTab,
    setPublicFollowActiveTab,
    publicFollowing,
    publicFollowers,
    publicFollowLoading,
    publicFollowError,
  };
}