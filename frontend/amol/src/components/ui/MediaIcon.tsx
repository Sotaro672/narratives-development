// frontend/amol/src/components/ui/MediaIcon.tsx
import type { ReactNode } from "react";

import "./media-icon.css";

type MediaIconSize = "xs" | "sm" | "md" | "lg";

type MediaIconShape = "circle" | "rounded";

type MediaIconProps = {
  src?: string | null;
  alt?: string;
  fallback?: ReactNode;
  size?: MediaIconSize;
  shape?: MediaIconShape;
  className?: string;
};

export default function MediaIcon(props: MediaIconProps) {
  const {
    src,
    alt = "",
    fallback = "img",
    size = "md",
    shape = "circle",
    className,
  } = props;

  const classes = [
    "ui-media-icon",
    `ui-media-icon--${size}`,
    `ui-media-icon--${shape}`,
    className,
  ]
    .filter(Boolean)
    .join(" ");

  if (src?.trim()) {
    return <img className={classes} src={src} alt={alt} />;
  }

  return (
    <div className={[classes, "ui-media-icon--empty"].join(" ")}>
      {fallback}
    </div>
  );
}