// frontend/mall/src/features/contents/hooks/useContentsSwipeDismiss.ts

import {
  useCallback,
  useRef,
} from "react";
import { useNavigate } from "react-router-dom";

import type { MobileSwipeDismissPageHandle } from "../../../components/layout/MobileSwipeDismissPage";

type UseContentsSwipeDismissParams = {
  onBeforeDismiss?: () => void;
};

export function useContentsSwipeDismiss({
  onBeforeDismiss,
}: UseContentsSwipeDismissParams = {}) {
  const navigate = useNavigate();

  const swipeDismissRef =
    useRef<MobileSwipeDismissPageHandle | null>(
      null,
    );

  const handleDismissStart =
    useCallback(() => {
      onBeforeDismiss?.();

      if (typeof document === "undefined") {
        return;
      }

      document
        .querySelectorAll<HTMLVideoElement>(
          "video",
        )
        .forEach((video) => {
          if (!video.paused) {
            video.pause();
          }
        });
    }, [
      onBeforeDismiss,
    ]);

  const handleDismiss =
    useCallback(() => {
      onBeforeDismiss?.();

      navigate("/wallet", {
        replace: true,
        state: {
          walletTab: "tokens",
        },
      });
    }, [
      navigate,
      onBeforeDismiss,
    ]);

  const handleDismissButtonClick =
    useCallback(() => {
      swipeDismissRef.current?.dismiss();
    }, []);

  return {
    swipeDismissRef,
    handleDismissStart,
    handleDismiss,
    handleDismissButtonClick,
  };
}