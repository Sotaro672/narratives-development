// frontend/shared/ui/label.tsx
import * as React from "react";

/** className helper */
function cn(...classes: Array<string | undefined | false | null>) {
  return classes.filter(Boolean).join(" ");
}

/**
 * Tailwind v4 + shared/index.css の HSLトークンを使った Label
 * - text-[hsl(var(--foreground))] でテーマに追従
 * - フォントウェイト/サイズは shadcn/ui 標準に準拠
 */
export interface LabelProps
  extends React.LabelHTMLAttributes<HTMLLabelElement> {}

export const Label = React.forwardRef<HTMLLabelElement, LabelProps>(
  ({ className, ...props }, ref) => (
    <label
      ref={ref}
      className={cn(
        "text-sm font-medium leading-none",
        "text-[hsl(var(--foreground))]",
        className
      )}
      {...props}
    />
  )
);

Label.displayName = "Label";
