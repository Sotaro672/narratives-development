// frontend/mall/src/features/contents/hooks/useContentsSwipeDismiss.ts

import { useCallback } from "react";
import { useNavigate } from "react-router-dom";

type UseContentsSwipeDismissParams = {
  onBeforeDismiss?: () => void;
};

export function useContentsSwipeDismiss({
  onBeforeDismiss,
}: UseContentsSwipeDismissParams = {}) {
  const navigate = useNavigate();

  const handleDismissStart = useCallback(() => {
    onBeforeDismiss?.();

    if (typeof document === "undefined") {
      return;
    }

    document
      .querySelectorAll<HTMLVideoElement>("video")
      .forEach((video) => {
        if (!video.paused) {
          video.pause();
        }
      });
  }, [onBeforeDismiss]);

  const handleDismiss = useCallback(() => {
    onBeforeDismiss?.();

    navigate("/wallet", {
      replace: true,
      state: {
        walletTab: "tokens",
      },
    });
  }, [navigate, onBeforeDismiss]);

  return {
    handleDismissStart,
    handleDismiss,
  };
}