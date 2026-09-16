// frontend/mall/src/components/hooks/useMobilePortrait.ts

import { useEffect, useState } from "react";

export const MOBILE_PORTRAIT_MEDIA_QUERY =
  "(max-width: 959px)";

type LegacyOrientationWindow = Window & {
  orientation?: number;
};

function isPhysicalPortrait(): boolean {
  if (typeof window === "undefined") {
    return false;
  }

  const legacyOrientation =
    (window as LegacyOrientationWindow).orientation;

  if (typeof legacyOrientation === "number") {
    return Math.abs(legacyOrientation) !== 90;
  }

  const orientationType =
    window.screen.orientation?.type;

  if (orientationType) {
    return orientationType.startsWith("portrait");
  }

  return window.screen.height >= window.screen.width;
}

export function useMobilePortrait(): boolean {
  const [isMobilePortrait, setIsMobilePortrait] =
    useState(false);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    const mediaQuery = window.matchMedia(
      MOBILE_PORTRAIT_MEDIA_QUERY,
    );

    const updateMobilePortraitState = () => {
      setIsMobilePortrait(
        mediaQuery.matches && isPhysicalPortrait(),
      );
    };

    updateMobilePortraitState();

    if (typeof mediaQuery.addEventListener === "function") {
      mediaQuery.addEventListener(
        "change",
        updateMobilePortraitState,
      );
    } else {
      mediaQuery.addListener(updateMobilePortraitState);
    }

    const screenOrientation =
      window.screen.orientation;

    if (
      screenOrientation &&
      typeof screenOrientation.addEventListener === "function"
    ) {
      screenOrientation.addEventListener(
        "change",
        updateMobilePortraitState,
      );
    }

    window.addEventListener(
      "orientationchange",
      updateMobilePortraitState,
    );

    return () => {
      if (
        typeof mediaQuery.removeEventListener === "function"
      ) {
        mediaQuery.removeEventListener(
          "change",
          updateMobilePortraitState,
        );
      } else {
        mediaQuery.removeListener(
          updateMobilePortraitState,
        );
      }

      if (
        screenOrientation &&
        typeof screenOrientation.removeEventListener ===
          "function"
      ) {
        screenOrientation.removeEventListener(
          "change",
          updateMobilePortraitState,
        );
      }

      window.removeEventListener(
        "orientationchange",
        updateMobilePortraitState,
      );
    };
  }, []);

  return isMobilePortrait;
}