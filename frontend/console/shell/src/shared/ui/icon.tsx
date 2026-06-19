// frontend/console/shell/src/shared/ui/icon.tsx
import * as React from "react";
import "./icon.css";

function cn(...classes: Array<string | undefined | false | null>) {
  return classes.filter(Boolean).join(" ");
}

export type AvatarIconSize = "sm" | "md" | "lg";

export type AvatarIconProps = {
  src?: string | null;
  name?: string | null;
  alt?: string;
  size?: AvatarIconSize;
  className?: string;
};

function getInitial(name?: string | null): string {
  const value = String(name ?? "");
  if (!value) return "-";

  return value.slice(0, 1).toUpperCase();
}

export default function AvatarIcon({
  src,
  name,
  alt,
  size = "md",
  className,
}: AvatarIconProps) {
  const imageUrl = String(src ?? "");
  const displayName = String(name ?? "");
  const label = alt ?? displayName ?? "avatar icon";

  return (
    <div
      className={cn(
        "avatar-icon",
        `avatar-icon--${size}`,
        className,
      )}
      aria-label={label}
    >
      {imageUrl ? (
        <img
          src={imageUrl}
          alt={label}
          className="avatar-icon__image"
          loading="lazy"
        />
      ) : (
        <div className="avatar-icon__fallback" aria-hidden="true">
          {getInitial(displayName)}
        </div>
      )}
    </div>
  );
}

export { AvatarIcon };