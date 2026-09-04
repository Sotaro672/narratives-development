// frontend/src/features/contact/hooks/useContactViewport.ts
import { useEffect, useState } from "react";

export function useContactViewport() {
  const [isDesktop, setIsDesktop] = useState(() => {
    if (typeof window === "undefined") {
      return false;
    }

    return window.matchMedia("(min-width: 1024px)").matches;
  });

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    const desktopQuery = window.matchMedia("(min-width: 1024px)");

    const updateDesktopState = () => {
      setIsDesktop(desktopQuery.matches);
    };

    updateDesktopState();

    if (typeof desktopQuery.addEventListener === "function") {
      desktopQuery.addEventListener("change", updateDesktopState);

      return () => {
        desktopQuery.removeEventListener("change", updateDesktopState);
      };
    }

    desktopQuery.addListener(updateDesktopState);

    return () => {
      desktopQuery.removeListener(updateDesktopState);
    };
  }, []);

  return {
    isDesktop,
  };
}