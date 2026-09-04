// frontend/amol/src/components/ui/TextState.tsx
import type { ReactNode } from "react";

import "./text-state.css";

type TextStateVariant = "muted" | "error" | "success" | "empty" | "loading";

type TextStateProps = {
  children: ReactNode;
  variant?: TextStateVariant;
  className?: string;
};

export default function TextState(props: TextStateProps) {
  const { children, variant = "muted", className } = props;

  return (
    <p
      className={["ui-text-state", `ui-text-state--${variant}`, className]
        .filter(Boolean)
        .join(" ")}
    >
      {children}
    </p>
  );
}