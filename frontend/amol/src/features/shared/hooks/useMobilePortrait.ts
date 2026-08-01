// frontend/amol/src/features/shared/hooks/useMobilePortrait.ts

import { useEffect, useState } from "react";

export const MOBILE_PORTRAIT_MEDIA_QUERY =
  "(max-width: 959px) and (orientation: portrait)";

export function useMobilePortrait(): boolean {
  const [isMobilePortrait, setIsMobilePortrait] = useState(false);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    const mediaQuery = window.matchMedia(MOBILE_PORTRAIT_MEDIA_QUERY);

    const updateMobilePortraitState = () => {
      setIsMobilePortrait(mediaQuery.matches);
    };

    updateMobilePortraitState();

    if (typeof mediaQuery.addEventListener === "function") {
      mediaQuery.addEventListener(
        "change",
        updateMobilePortraitState,
      );

      return () => {
        mediaQuery.removeEventListener(
          "change",
          updateMobilePortraitState,
        );
      };
    }

    mediaQuery.addListener(updateMobilePortraitState);

    return () => {
      mediaQuery.removeListener(updateMobilePortraitState);
    };
  }, []);

  return isMobilePortrait;
}