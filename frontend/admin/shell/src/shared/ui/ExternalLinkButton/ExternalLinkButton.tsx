// frontend/admin/shell/src/shared/ui/ExternalLinkButton/ExternalLinkButton.tsx

import type { ReactNode } from "react";

import "./ExternalLinkButton.css";

type ExternalLinkButtonProps = {
  href: string;
  children: ReactNode;
  title?: string;
  ariaLabel?: string;
};

export default function ExternalLinkButton({
  href,
  children,
  title,
  ariaLabel,
}: ExternalLinkButtonProps) {
  return (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      className="ui-external-link-button"
      title={title}
      aria-label={ariaLabel}
    >
      {children}
    </a>
  );
}