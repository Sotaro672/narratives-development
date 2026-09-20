// frontend/mall/src/components/ui/Card.tsx

import {
  type HTMLAttributes,
  type KeyboardEvent,
  type ReactNode,
} from "react";

import "./Card.css";

type CardPadding = "sm" | "md" | "lg";
type CardElement = "div" | "article" | "section";

type CardProps = HTMLAttributes<HTMLElement> & {
  children: ReactNode;
  as?: CardElement;
  interactive?: boolean;
  highlighted?: boolean;
  busy?: boolean;
  padding?: CardPadding;
};

export default function Card({
  children,
  as: Component = "div",
  interactive = false,
  highlighted = false,
  busy = false,
  padding = "md",
  className = "",
  tabIndex,
  role,
  onClick,
  onKeyDown,
  ...props
}: CardProps) {
  const classes = [
    "ui-card",
    `ui-card--padding-${padding}`,
    interactive ? "ui-card--interactive" : "",
    highlighted ? "ui-card--highlighted" : "",
    busy ? "ui-card--busy" : "",
    className,
  ]
    .filter(Boolean)
    .join(" ");

  const handleKeyDown = (event: KeyboardEvent<HTMLElement>) => {
    onKeyDown?.(event);

    if (
      event.defaultPrevented ||
      !interactive ||
      busy ||
      !onClick ||
      (event.key !== "Enter" && event.key !== " ")
    ) {
      return;
    }

    event.preventDefault();
    event.currentTarget.click();
  };

  return (
    <Component
      className={classes}
      role={role ?? (interactive ? "button" : undefined)}
      tabIndex={tabIndex ?? (interactive && !busy ? 0 : undefined)}
      aria-busy={busy || undefined}
      onClick={onClick}
      onKeyDown={handleKeyDown}
      {...props}
    >
      {children}
    </Component>
  );
}