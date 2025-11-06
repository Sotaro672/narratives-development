// frontend/shared/ui/card.tsx
import * as React from "react";

/**
 * cn() — tailwind + condition helper
 */
function cn(...classes: Array<string | undefined | false | null>) {
  return classes.filter(Boolean).join(" ");
}

/**
 * Card コンポーネント群
 * Tailwind v4 + shared/index.css の HSL トークンに対応
 */
export const Card = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
  <div
    ref={ref}
    className={cn(
      // Tailwind v4 utility + shared tokens
      "rounded-xl border bg-[hsl(var(--card))] text-[hsl(var(--card-foreground))]",
      "border-[hsl(var(--border))] shadow-sm",
      "p-0", // 内側余白は Header/Content 側で付与
      "m-[3px]", // ← 4辺に3pxのmarginを付与
      className
    )}
    {...props}
  />
));
Card.displayName = "Card";

/**
 * CardHeader
 */
export const CardHeader = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
  <div
    ref={ref}
    className={cn(
      "px-5 py-4 border-b",
      "border-[hsl(var(--border))]",
      "bg-[hsl(var(--card))]",
      className
    )}
    {...props}
  />
));
CardHeader.displayName = "CardHeader";

/**
 * CardTitle
 */
export const CardTitle = React.forwardRef<
  HTMLHeadingElement,
  React.HTMLAttributes<HTMLHeadingElement>
>(({ className, ...props }, ref) => (
  <h3
    ref={ref}
    className={cn(
      "text-base font-semibold leading-none tracking-tight",
      "text-[hsl(var(--foreground))]",
      className
    )}
    {...props}
  />
));
CardTitle.displayName = "CardTitle";

/**
 * CardContent
 */
export const CardContent = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
  <div
    ref={ref}
    className={cn(
      "px-5 py-4",
      "text-[hsl(var(--card-foreground))]",
      className
    )}
    {...props}
  />
));
CardContent.displayName = "CardContent";
