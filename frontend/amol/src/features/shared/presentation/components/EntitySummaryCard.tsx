// frontend/amol/src/features/shared/presentation/components/EntitySummaryCard.tsx

import type { ReactNode } from "react";

import MediaIcon from "../../../../components/ui/MediaIcon";

import "../../styles/entity-summary-card.css";

export type EntitySummaryCardProps = {
  icon?: string | null;
  iconAlt?: string;
  iconFallback?: ReactNode;
  label?: string | null;
  name?: string | null;
  secondaryText?: string | null;
  description?: string | null;
  onClick?: () => void;
  disabled?: boolean;
  className?: string;
};

function joinClassNames(...classNames: Array<string | undefined | false>): string {
  return classNames.filter(Boolean).join(" ");
}

export default function EntitySummaryCard({
  icon,
  iconAlt,
  iconFallback = "◎",
  label,
  name,
  secondaryText,
  description,
  onClick,
  disabled = false,
  className,
}: EntitySummaryCardProps) {
  const safeIcon = icon?.trim() || "";
  const safeLabel = label?.trim() || "";
  const safeName = name?.trim() || "";
  const safeSecondaryText = secondaryText?.trim() || "";
  const safeDescription = description?.trim() || "";
  const isInteractive = Boolean(onClick);

  if (!safeIcon && !safeLabel && !safeName && !safeSecondaryText && !safeDescription) {
    return null;
  }

  const content = (
    <>
      <MediaIcon
        src={safeIcon}
        alt={iconAlt?.trim() || safeName || safeLabel || "アイコン"}
        fallback={iconFallback}
        size="lg"
        shape="rounded"
        className="entity-summary-card__icon"
      />

      <div className="entity-summary-card__body">
        {safeLabel ? <span className="entity-summary-card__label">{safeLabel}</span> : null}
        {safeName ? <span className="entity-summary-card__name">{safeName}</span> : null}
        {safeSecondaryText ? (
          <span className="entity-summary-card__secondary">{safeSecondaryText}</span>
        ) : null}
        {safeDescription ? (
          <p className="entity-summary-card__description">{safeDescription}</p>
        ) : null}
      </div>

      {isInteractive && !disabled ? (
        <span className="entity-summary-card__arrow" aria-hidden="true">
          ›
        </span>
      ) : null}
    </>
  );

  if (isInteractive) {
    return (
      <button
        type="button"
        className={joinClassNames("entity-summary-card", "entity-summary-card--button", className)}
        onClick={onClick}
        disabled={disabled}
      >
        {content}
      </button>
    );
  }

  return <div className={joinClassNames("entity-summary-card", className)}>{content}</div>;
}