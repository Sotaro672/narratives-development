// frontend/amol/src/features/shared/presentation/components/EntitySummaryCard.tsx

import {
  useId,
  useState,
  type ReactNode,
} from "react";

import MediaIcon from "../../../../components/ui/MediaIcon";

import "../../styles/entity-summary-card.css";

const DESCRIPTION_COLLAPSE_THRESHOLD = 80;

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
  const descriptionId = useId();
  const [descriptionExpanded, setDescriptionExpanded] = useState(false);

  const safeIcon = icon?.trim() || "";
  const safeLabel = label?.trim() || "";
  const safeName = name?.trim() || "";
  const safeSecondaryText = secondaryText?.trim() || "";
  const safeDescription = description?.trim() || "";
  const isInteractive = Boolean(onClick);
  const isDescriptionExpandable = safeDescription.length > DESCRIPTION_COLLAPSE_THRESHOLD;

  if (!safeIcon && !safeLabel && !safeName && !safeSecondaryText && !safeDescription) {
    return null;
  }

  const summaryContent = (
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
      </div>

      {isInteractive && !disabled ? (
        <span className="entity-summary-card__arrow" aria-hidden="true">
          ›
        </span>
      ) : null}
    </>
  );

  return (
    <div
      className={joinClassNames(
        "entity-summary-card",
        isInteractive && "entity-summary-card--interactive",
        disabled && "entity-summary-card--disabled",
        className,
      )}
    >
      {isInteractive ? (
        <button
          type="button"
          className="entity-summary-card__main entity-summary-card__main--button"
          onClick={onClick}
          disabled={disabled}
        >
          {summaryContent}
        </button>
      ) : (
        <div className="entity-summary-card__main">
          {summaryContent}
        </div>
      )}

      {safeDescription ? (
        <div className="entity-summary-card__description-area">
          <p
            id={descriptionId}
            className={joinClassNames(
              "entity-summary-card__description",
              isDescriptionExpandable &&
                !descriptionExpanded &&
                "entity-summary-card__description--collapsed",
            )}
          >
            {safeDescription}
          </p>

          {isDescriptionExpandable ? (
            <button
              type="button"
              className="entity-summary-card__description-toggle"
              aria-expanded={descriptionExpanded}
              aria-controls={descriptionId}
              onClick={() => setDescriptionExpanded((current) => !current)}
            >
              {descriptionExpanded ? "閉じる" : "詳しく見る"}
            </button>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}