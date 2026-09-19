// frontend/console/shell/src/shared/ui/empty.tsx

import * as React from "react";

import "./empty.css";

function cn(...classes: Array<string | undefined | false | null>): string {
  return classes.filter(Boolean).join(" ");
}

export type EmptyProps = React.HTMLAttributes<HTMLDivElement> & {
  title?: React.ReactNode;
  description?: React.ReactNode;
  compact?: boolean;
};

export const Empty = React.forwardRef<HTMLDivElement, EmptyProps>(
  (
    {
      title,
      description = "現在登録されている項目はございません。",
      compact = false,
      className,
      children,
      ...props
    },
    ref,
  ) => (
    <div
      ref={ref}
      className={cn(
        "empty",
        compact && "empty--compact",
        className,
      )}
      {...props}
    >
      {title ? (
        <div className="empty__title">
          {title}
        </div>
      ) : null}

      {description ? (
        <div className="empty__description">
          {description}
        </div>
      ) : null}

      {children ? (
        <div className="empty__content">
          {children}
        </div>
      ) : null}
    </div>
  ),
);

Empty.displayName = "Empty";

export default Empty;