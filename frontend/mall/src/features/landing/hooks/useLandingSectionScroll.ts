// frontend/mall/src/features/landing/hooks/useLandingSectionScroll.ts

import { useCallback, useEffect, useRef } from "react";
import type { RefObject } from "react";
import { useLocation } from "react-router-dom";

export type UseLandingSectionScrollResult = {
  authenticationEyebrowRef: RefObject<HTMLParagraphElement>;
  salesSupportEyebrowRef: RefObject<HTMLParagraphElement>;
  fleaMarketEyebrowRef: RefObject<HTMLParagraphElement>;
  scrollToAuthentication: () => void;
  scrollToSalesSupport: () => void;
  scrollToFleaMarket: () => void;
};

export function useLandingSectionScroll(): UseLandingSectionScrollResult {
  const location = useLocation();

  const authenticationEyebrowRef = useRef<HTMLParagraphElement>(null);
  const salesSupportEyebrowRef = useRef<HTMLParagraphElement>(null);
  const fleaMarketEyebrowRef = useRef<HTMLParagraphElement>(null);

  const scrollToElement = useCallback((element: HTMLElement | null) => {
    if (!element || typeof window === "undefined") {
      return;
    }

    const isMobileViewport = window.matchMedia("(max-width: 767px)").matches;
    const headerOffset = isMobileViewport ? 72 : 88;

    const elementTop =
      window.scrollY +
      element.getBoundingClientRect().top -
      headerOffset;

    window.scrollTo({
      top: Math.max(0, elementTop),
      behavior: "smooth",
    });
  }, []);

  const scrollToAuthentication = useCallback(() => {
    scrollToElement(authenticationEyebrowRef.current);
  }, [scrollToElement]);

  const scrollToSalesSupport = useCallback(() => {
    scrollToElement(salesSupportEyebrowRef.current);
  }, [scrollToElement]);

  const scrollToFleaMarket = useCallback(() => {
    scrollToElement(fleaMarketEyebrowRef.current);
  }, [scrollToElement]);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    const sectionId = location.hash.replace(/^#/, "").trim();

    if (!sectionId) {
      return;
    }

    let firstFrameId = 0;
    let secondFrameId = 0;

    firstFrameId = window.requestAnimationFrame(() => {
      secondFrameId = window.requestAnimationFrame(() => {
        const section = document.getElementById(sectionId);
        scrollToElement(section);
      });
    });

    return () => {
      window.cancelAnimationFrame(firstFrameId);
      window.cancelAnimationFrame(secondFrameId);
    };
  }, [location.hash, location.key, scrollToElement]);

  return {
    authenticationEyebrowRef,
    salesSupportEyebrowRef,
    fleaMarketEyebrowRef,
    scrollToAuthentication,
    scrollToSalesSupport,
    scrollToFleaMarket,
  };
}

export default useLandingSectionScroll;