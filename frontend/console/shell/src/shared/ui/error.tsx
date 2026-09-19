// frontend/console/shell/src/shared/ui/error.tsx

import * as React from "react";

import {
  Text,
  type TextProps,
} from "./text";

import "./error.css";

function cn(...classes: Array<string | undefined | false | null>): string {
  return classes.filter(Boolean).join(" ");
}

export type ErrorMessageVariant =
  | "text"
  | "panel";

export type ErrorMessageProps = Omit<TextProps, "tone"> & {
  variant?: ErrorMessageVariant;
};

export const ErrorMessage = React.forwardRef<
  HTMLElement,
  ErrorMessageProps
>(
  (
    {
      as = "div",
      size = "sm",
      weight = "normal",
      wrap = "pre-wrap",
      variant = "text",
      className,
      role = "alert",
      ...props
    },
    ref,
  ) => (
    <Text
      ref={ref}
      as={as}
      size={size}
      tone="destructive"
      weight={weight}
      wrap={wrap}
      role={role}
      className={cn(
        "error-message",
        variant === "panel" && "error-message--panel",
        className,
      )}
      {...props}
    />
  ),
);

ErrorMessage.displayName = "ErrorMessage";

export default ErrorMessage;