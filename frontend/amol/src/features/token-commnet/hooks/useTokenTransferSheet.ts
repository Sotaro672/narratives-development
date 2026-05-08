// frontend/amol/src/features/token-commnet/hooks/useTokenTransferSheet.ts
import { useCallback, useMemo, useState } from "react";

import {
  fetchTokenTransferFollowState,
  getCurrentIdToken,
  transferTokenToAvatar,
} from "../api/tokenTransferApi";
import type {
  TokenTransferFollowState,
  TokenTransferSheetState,
  TokenTransferTargetTabKey,
  TransferTokenToAvatarResponse,
} from "../types/tokenTransferTypes";

type UseTokenTransferSheetParams = {
  productId: string;
  currentAvatarId: string;
  initialTab?: TokenTransferTargetTabKey;
  onTransferred?: (
    response: TransferTokenToAvatarResponse
  ) => void | Promise<void>;
};

const BACKEND_BASE_URL = import.meta.env.VITE_API_BASE_URL;

export function useTokenTransferSheet({
  productId,
  currentAvatarId,
  initialTab = "following",
  onTransferred,
}: UseTokenTransferSheetParams) {
  const [open, setOpen] = useState(false);
  const [activeTab, setActiveTab] =
    useState<TokenTransferTargetTabKey>(initialTab);
  const [followState, setFollowState] =
    useState<TokenTransferFollowState | null>(null);
  const [selectedTargetAvatarId, setSelectedTargetAvatarId] = useState("");

  const [loading, setLoading] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");

  const state: TokenTransferSheetState = useMemo(
    () => ({
      open,
      activeTab,
      loading,
      refreshing,
      submitting,
      errorMessage,
      selectedTargetAvatarId,
      followState,
    }),
    [
      activeTab,
      errorMessage,
      followState,
      loading,
      open,
      refreshing,
      selectedTargetAvatarId,
      submitting,
    ]
  );

  const loadFollowState = useCallback(
    async (options?: { silent?: boolean }) => {
      const avatarId = currentAvatarId.trim();

      if (!avatarId) {
        setErrorMessage("アバターIDを取得できませんでした。");
        return;
      }

      if (!BACKEND_BASE_URL) {
        setErrorMessage("VITE_API_BASE_URL is not configured.");
        return;
      }

      if (options?.silent) {
        setRefreshing(true);
      } else {
        setLoading(true);
      }

      setErrorMessage("");

      try {
        const idToken = await getCurrentIdToken();

        const nextFollowState = await fetchTokenTransferFollowState({
          backendUrl: BACKEND_BASE_URL,
          idToken,
          avatarId,
        });

        setFollowState(nextFollowState);
      } catch (error) {
        setErrorMessage(
          error instanceof Error
            ? error.message
            : "フォロー情報の取得に失敗しました。"
        );
      } finally {
        setLoading(false);
        setRefreshing(false);
      }
    },
    [currentAvatarId]
  );

  const openSheet = useCallback(async () => {
    setOpen(true);
    setSelectedTargetAvatarId("");
    setErrorMessage("");

    if (!followState) {
      await loadFollowState();
    }
  }, [followState, loadFollowState]);

  const closeSheet = useCallback(() => {
    if (submitting) {
      return;
    }

    setOpen(false);
  }, [submitting]);

  const changeTab = useCallback((tab: TokenTransferTargetTabKey) => {
    setActiveTab(tab);
    setSelectedTargetAvatarId("");
  }, []);

  const refresh = useCallback(async () => {
    await loadFollowState({ silent: true });
  }, [loadFollowState]);

  const selectTarget = useCallback((targetAvatarId: string) => {
    setSelectedTargetAvatarId((current) =>
      current === targetAvatarId ? "" : targetAvatarId
    );
  }, []);

  const submit = useCallback(async () => {
    if (!BACKEND_BASE_URL) {
      setErrorMessage("VITE_API_BASE_URL is not configured.");
      return;
    }

    const normalizedProductId = productId.trim();
    const normalizedTargetAvatarId = selectedTargetAvatarId.trim();

    if (!normalizedProductId) {
      setErrorMessage("productId が空です。");
      return;
    }

    if (!normalizedTargetAvatarId) {
      setErrorMessage("渡す相手を選択してください。");
      return;
    }

    setSubmitting(true);
    setErrorMessage("");

    try {
      const idToken = await getCurrentIdToken();

      const result = await transferTokenToAvatar({
        backendUrl: BACKEND_BASE_URL,
        idToken,
        productId: normalizedProductId,
        targetAvatarId: normalizedTargetAvatarId,
      });

      await onTransferred?.(result);

      setOpen(false);
      setSelectedTargetAvatarId("");
    } catch (error) {
      setErrorMessage(
        error instanceof Error ? error.message : "トークンの移譲に失敗しました。"
      );
    } finally {
      setSubmitting(false);
    }
  }, [onTransferred, productId, selectedTargetAvatarId]);

  return {
    state,
    openSheet,
    closeSheet,
    changeTab,
    refresh,
    selectTarget,
    submit,
  };
}