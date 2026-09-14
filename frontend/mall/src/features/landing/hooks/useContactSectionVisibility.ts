// frontend/mall/src/features/landing/hooks/useContactSectionVisibility.ts

import { useEffect, useRef, useState } from "react";
import type { RefObject } from "react";

export type UseContactSectionVisibilityResult = {
  contactSectionRef: RefObject<HTMLElement>;
  isContactSectionVisible: boolean;
};

type UseContactSectionVisibilityOptions = {
  isDesktop: boolean;
};

export function useContactSectionVisibility({
  isDesktop,
}: UseContactSectionVisibilityOptions): UseContactSectionVisibilityResult {
  const contactSectionRef = useRef<HTMLElement>(null);
  const [isContactSectionVisible, setIsContactSectionVisible] =
    useState(false);

  useEffect(() => {
    if (typeof window === "undefined" || isDesktop) {
      setIsContactSectionVisible(false);
      return;
    }

    const contactSection = contactSectionRef.current;

    if (!contactSection) {
      return;
    }

    const observer = new IntersectionObserver(
      ([entry]) => {
        setIsContactSectionVisible(entry.isIntersecting);
      },
      {
        threshold: 0.1,
      },
    );

    observer.observe(contactSection);

    return () => {
      observer.disconnect();
    };
  }, [isDesktop]);

  return {
    contactSectionRef,
    isContactSectionVisible,
  };
}

export default useContactSectionVisibility;