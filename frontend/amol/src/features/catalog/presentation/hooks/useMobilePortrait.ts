//frontend\amol\src\features\catalog\hooks\useMobilePortrait.ts
import { useEffect, useState } from "react";

import { MOBILE_PORTRAIT_MEDIA_QUERY } from "../../constants";

export function useMobilePortrait(): boolean {
  const [isMobilePortrait, setIsMobilePortrait] = useState(false);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    const mobilePortraitQuery = window.matchMedia(MOBILE_PORTRAIT_MEDIA_QUERY);

    const updateMobilePortraitState = () => {
      setIsMobilePortrait(mobilePortraitQuery.matches);
    };

    updateMobilePortraitState();

    if (typeof mobilePortraitQuery.addEventListener === "function") {
      mobilePortraitQuery.addEventListener(
        "change",
        updateMobilePortraitState,
      );

      return () => {
        mobilePortraitQuery.removeEventListener(
          "change",
          updateMobilePortraitState,
        );
      };
    }

    mobilePortraitQuery.addListener(updateMobilePortraitState);

    return () => {
      mobilePortraitQuery.removeListener(updateMobilePortraitState);
    };
  }, []);

  return isMobilePortrait;
}