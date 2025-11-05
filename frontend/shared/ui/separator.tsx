// frontend/shared/ui/separator.tsx
import * as React from "react";

/** className helper */
function cn(...classes: Array<string | undefined | false | null>) {
  return classes.filter(Boolean).join(" ");
}

/**
 * Tailwind v4 + shared/index.css トークン前提の Separator
 * - hsl(var(--border)) でボーダーカラーを統一
 * - orientation に応じて horizontal / vertical を切替
 */
export interface SeparatorProps
  extends React.HTMLAttributes<HTMLDivElement> {
  orientation?: "horizontal" | "vertical";
}

export const Separator = React.forwardRef<HTMLDivElement, SeparatorProps>(
  ({ className, orientation = "horizontal", ...props }, ref) => (
    <div
      ref={ref}
      role="separator"
      className={cn(
        "shrink-0 bg-[hsl(var(--border))]",
        orientation === "horizontal" ? "h-px w-full" : "w-px h-full",
        className
      )}
      {...props}
    />
  )
);

Separator.displayName = "Separator";
