// frontend/mall/src/features/landing/hooks/useLandingViewport.ts

import { useEffect, useState } from "react";

export type UseLandingViewportResult = {
  isMobile: boolean;
  isDesktop: boolean;
};

export function useLandingViewport(): UseLandingViewportResult {
  const [isMobile, setIsMobile] = useState(false);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    const mediaQuery = window.matchMedia(
      "(max-width: 1023px)",
    );

    const updateViewportState = () => {
      setIsMobile(mediaQuery.matches);
    };

    updateViewportState();

    if (
      typeof mediaQuery.addEventListener ===
      "function"
    ) {
      mediaQuery.addEventListener(
        "change",
        updateViewportState,
      );

      return () => {
        mediaQuery.removeEventListener(
          "change",
          updateViewportState,
        );
      };
    }

    mediaQuery.addListener(
      updateViewportState,
    );

    return () => {
      mediaQuery.removeListener(
        updateViewportState,
      );
    };
  }, []);

  return {
    isMobile,
    isDesktop: !isMobile,
  };
}

export default useLandingViewport;