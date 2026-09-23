// frontend/console/shell/src/shared/ui/loading.tsx

import * as React from "react";
import { LoaderCircle } from "lucide-react";

import "./loading.css";

function cn(...classes: Array<string | undefined | false | null>): string {
  return classes.filter(Boolean).join(" ");
}

export type LoadingVariant =
  | "page"
  | "card"
  | "inline"
  | "table";

export type LoadingProps = React.HTMLAttributes<HTMLDivElement> & {
  message?: React.ReactNode;
  variant?: LoadingVariant;
  showIndicator?: boolean;
};

export const Loading = React.forwardRef<HTMLDivElement, LoadingProps>(
  (
    {
      message = "読み込み中...",
      variant = "inline",
      showIndicator = true,
      className,
      children,
      ...props
    },
    ref,
  ) => (
    <div
      ref={ref}
      role="status"
      aria-live="polite"
      aria-busy="true"
      className={cn(
        "loading",
        `loading--${variant}`,
        className,
      )}
      {...props}
    >
      {showIndicator ? (
        <LoaderCircle
          className="loading__indicator"
          aria-hidden="true"
        />
      ) : null}

      <div className="loading__message">
        {children ?? message}
      </div>
    </div>
  ),
);

Loading.displayName = "Loading";

export default Loading;