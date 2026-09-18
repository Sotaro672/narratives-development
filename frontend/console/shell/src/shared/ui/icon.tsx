// frontend/console/shell/src/shared/ui/icon.tsx

import * as React from "react";

import "./icon.css";

function cn(...classes: Array<string | undefined | false | null>): string {
  return classes.filter(Boolean).join(" ");
}

export type EntityIconSize = "sm" | "md" | "lg" | "fluid";

export type EntityIconProps = {
  src?: string | null;
  name?: string | null;
  alt?: string;
  size?: EntityIconSize;
  className?: string;
  imageClassName?: string;
  fallbackClassName?: string;
  fallback?: React.ReactNode;
  loading?: "eager" | "lazy";
  onClick?: () => void;
  disabled?: boolean;
};

function getInitial(name?: string | null): string {
  const value = String(name ?? "").trim();

  if (!value) {
    return "-";
  }

  return value.slice(0, 1).toUpperCase();
}

export default function EntityIcon({
  src,
  name,
  alt,
  size = "md",
  className,
  imageClassName,
  fallbackClassName,
  fallback,
  loading = "lazy",
  onClick,
  disabled = false,
}: EntityIconProps) {
  const imageUrl = String(src ?? "").trim();
  const displayName = String(name ?? "").trim();
  const label = alt || displayName || "アイコン";

  const content = imageUrl ? (
    <img
      src={imageUrl}
      alt={label}
      className={cn("entity-icon__image", imageClassName)}
      loading={loading}
    />
  ) : (
    <span
      className={cn("entity-icon__fallback", fallbackClassName)}
      aria-hidden="true"
    >
      {fallback ?? getInitial(displayName)}
    </span>
  );

  const rootClassName = cn(
    "entity-icon",
    `entity-icon--${size}`,
    onClick && "entity-icon--interactive",
    disabled && "entity-icon--disabled",
    className,
  );

  if (onClick) {
    return (
      <button
        type="button"
        className={rootClassName}
        onClick={onClick}
        disabled={disabled}
        aria-label={label}
      >
        {content}
      </button>
    );
  }

  return (
    <span
      className={rootClassName}
      aria-label={label}
      role="img"
    >
      {content}
    </span>
  );
}

export { EntityIcon };