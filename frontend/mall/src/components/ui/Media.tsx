// frontend/mall/src/components/ui/Media.tsx

import type { ImgHTMLAttributes } from "react";

import "./media.css";

type MediaProps = ImgHTMLAttributes<HTMLImageElement> & {
  src: string;
  alt: string;
  fit?: "cover" | "contain";
};

export default function Media({
  src,
  alt,
  fit = "cover",
  className = "",
  ...props
}: MediaProps) {
  const classes = [
    "ui-media",
    `ui-media--${fit}`,
    className,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <img
      src={src}
      alt={alt}
      className={classes}
      {...props}
    />
  );
}