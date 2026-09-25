// frontend/mall/src/components/ui/textLink.tsx

import type { ComponentProps } from "react";

import TextButton from "./TextButton";
import "./textLink.css";

type TextLinkProps = ComponentProps<typeof TextButton>;

export default function TextLink({
  className = "",
  ...props
}: TextLinkProps) {
  const classes = ["text-link", className].filter(Boolean).join(" ");

  return <TextButton className={classes} {...props} />;
}