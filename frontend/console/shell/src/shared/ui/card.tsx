// frontend\shell\src\shared\ui\card.tsx
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
 *
 * 旧 .vc / .mnc / .svc 共通スタイルをここに集約:
 * - 外枠: Card
 * - ヘッダー: CardHeader + CardTitle
 * - 本文: CardContent
 * - ラベル/入力など: CardLabel / CardInput / CardSelect / CardReadonly / CardSuffix
 * - チップ/ボタン: CardChips / CardChip / CardButton
 */

export const Card = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
  <div
    ref={ref}
    className={cn(
      // ベース外観（pbp-surface / pbp-border 相当）
      "rounded-xl border bg-[hsl(var(--card))] text-[hsl(var(--card-foreground))]",
      "border-[hsl(var(--border))] shadow-sm",
      // 旧 .vc/.mnc/.svc の margin: 3px 0px;
      "my-[3px]",
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
      // 旧 box__header: flex + padding + bottom border
      "flex items-center gap-2 px-5 py-4 border-b",
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
      // 旧 box__title: 0.875rem / 600 / dim color
      "text-[0.875rem] font-semibold leading-none tracking-tight",
      "text-[hsl(var(--muted-foreground))]",
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
      // 旧 box__body: padding:20px 相当
      "px-5 py-5 text-[hsl(var(--card-foreground))]",
      className
    )}
    {...props}
  />
));
CardContent.displayName = "CardContent";

/**
 * CardFooter（必要なら使用）
 */
export const CardFooter = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
  <div
    ref={ref}
    className={cn(
      "px-5 py-3 border-t border-[hsl(var(--border))] flex items-center justify-end gap-2",
      className
    )}
    {...props}
  />
));
CardFooter.displayName = "CardFooter";

/**
 * 共通フォーム系
 * 旧 .label / .input / .select / .readonly / .suffix
 */

export const CardLabel = React.forwardRef<
  HTMLLabelElement,
  React.LabelHTMLAttributes<HTMLLabelElement>
>(({ className, ...props }, ref) => (
  <label
    ref={ref}
    className={cn(
      // font-size:12px / margin-top:12px / margin-bottom:6px
      "mt-3 mb-1 block text-[0.75rem] text-[hsl(var(--muted-foreground))]",
      className
    )}
    {...props}
  />
));
CardLabel.displayName = "CardLabel";

export const CardInput = React.forwardRef<
  HTMLInputElement,
  React.InputHTMLAttributes<HTMLInputElement>
>(({ className, ...props }, ref) => (
  <input
    ref={ref}
    className={cn(
      // 旧 .input
      "h-10 w-full rounded-[10px] px-3 text-sm",
      "border border-[hsl(var(--border-strong,var(--border)))]",
      "bg-[hsl(var(--input-bg,var(--background)))]",
      "text-[hsl(var(--foreground))]",
      "outline-none focus:ring-2 focus:ring-[hsl(var(--ring))]",
      className
    )}
    {...props}
  />
));
CardInput.displayName = "CardInput";

export const CardSelect = React.forwardRef<
  HTMLSelectElement,
  React.SelectHTMLAttributes<HTMLSelectElement>
>(({ className, ...props }, ref) => (
  <select
    ref={ref}
    className={cn(
      // 旧 .select と同等
      "h-10 w-full rounded-[10px] px-3 text-sm",
      "border border-[hsl(var(--border-strong,var(--border)))]",
      "bg-[hsl(var(--input-bg,var(--background)))]",
      "text-[hsl(var(--foreground))]",
      "outline-none focus:ring-2 focus:ring-[hsl(var(--ring))]",
      className
    )}
    {...props}
  />
));
CardSelect.displayName = "CardSelect";

/**
 * 読み取り専用表示 (旧 .readonly)
 */
export const CardReadonly = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
  <div
    ref={ref}
    className={cn(
      "h-10 w-full rounded-[10px] px-3 text-sm",
      "flex items-center",
      "border border-[hsl(var(--border-soft,var(--border)))]",
      "bg-[hsl(var(--muted-bg,var(--muted)))]",
      "text-[hsl(var(--muted-foreground))]",
      "cursor-not-allowed",
      className
    )}
    {...props}
  />
));
CardReadonly.displayName = "CardReadonly";

/**
 * サフィックス (旧 .suffix)
 */
export const CardSuffix = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
  <div
    ref={ref}
    className={cn(
      "h-10 inline-flex items-center justify-center px-3",
      "rounded-[10px] border",
      "border-[hsl(var(--border-strong,var(--border)))]",
      "bg-[hsl(var(--input-bg,var(--background)))]",
      "text-[hsl(var(--muted-foreground))]",
      className
    )}
    {...props}
  />
));
CardSuffix.displayName = "CardSuffix";

/**
 * チップ系 (旧 .chips / .chip)
 */

export const CardChips = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
  <div
    ref={ref}
    className={cn(
      "flex flex-wrap gap-2 mb-2",
      className
    )}
    {...props}
  />
));
CardChips.displayName = "CardChips";

export const CardChip = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
  <div
    ref={ref}
    className={cn(
      // 旧 .chip
      "inline-flex items-center gap-1.5 h-7 px-2 rounded-full",
      "bg-[hsl(var(--chip-bg,var(--muted)))]",
      "text-[0.75rem] text-[hsl(var(--chip-fg,var(--muted-foreground)))]",
      className
    )}
    {...props}
  />
));
CardChip.displayName = "CardChip";

/**
 * ボタン (旧 .btn)
 */
export const CardButton = React.forwardRef<
  HTMLButtonElement,
  React.ButtonHTMLAttributes<HTMLButtonElement>
>(({ className, ...props }, ref) => (
  <button
    ref={ref}
    className={cn(
      "inline-flex items-center justify-center gap-1.5",
      "h-9 px-3 rounded-[10px]",
      "border border-[hsl(var(--border))]",
      "bg-[hsl(var(--card))] text-[hsl(var(--foreground))]",
      "text-sm cursor-pointer",
      "hover:bg-[hsl(var(--muted))]",
      className
    )}
    {...props}
  />
));
CardButton.displayName = "CardButton";
