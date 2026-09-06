// frontend/admin/shell/src/shared/ui/ExternalLinkButton/ExternalLinkButton.tsx

import type { ReactNode } from "react";

import { getButtonClassName } from "../Button/Button";

type ExternalLinkButtonProps = {
  href: string;
  children: ReactNode;
  title?: string;
  ariaLabel?: string;
  className?: string;
};

export default function ExternalLinkButton({
  href,
  children,
  title,
  ariaLabel,
  className = "",
}: ExternalLinkButtonProps) {
  return (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      className={getButtonClassName({
        variant: "secondary",
        size: "lg",
        className,
      })}
      title={title}
      aria-label={ariaLabel}
    >
      {children}
    </a>
  );
}