// frontend/shared/ui/button.tsx
import * as React from "react";

/** className helper */
function cn(...classes: Array<string | undefined | false | null>) {
  return classes.filter(Boolean).join(" ");
}

type Variant = "default" | "ghost" | "outline";
type Size = "sm" | "md";

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant;
  size?: Size;
}

/**
 * Tailwind v4 + shared/index.css の HSLトークン前提の Button
 * - 背景/文字/枠線は hsl(var(--...)) でテーマに追従
 * - @apply 不使用
 */
export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant = "default", size = "md", ...props }, ref) => {
    const base =
      "inline-flex items-center justify-center rounded-md transition-colors " +
      "focus-visible:outline-none focus-visible:ring-2 " +
      "focus-visible:ring-[hsl(var(--ring))]";

    const byVariant: Record<Variant, string> = {
      default:
        "bg-[hsl(var(--primary))] text-[hsl(var(--primary-foreground))] " +
        "border border-transparent hover:opacity-95",
      ghost:
        "bg-transparent text-[hsl(var(--foreground))] " +
        "border border-transparent hover:bg-[hsl(var(--muted))]",
      outline:
        "bg-[hsl(var(--card))] text-[hsl(var(--foreground))] " +
        "border border-[hsl(var(--border))] hover:bg-[hsl(var(--muted))]",
    };

    const bySize: Record<Size, string> = {
      sm: "h-8 px-2 text-sm",
      md: "h-10 px-3 text-sm",
    };

    return (
      <button
        ref={ref}
        className={cn(base, byVariant[variant], bySize[size], className)}
        {...props}
      />
    );
  }
);
Button.displayName = "Button";
