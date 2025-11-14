// frontend\shell\src\shared\ui\button.tsx
import * as React from "react";

/** className helper */
function cn(...classes: Array<string | undefined | false | null>) {
  return classes.filter(Boolean).join(" ");
}

/**
 * 簡易 Slot 実装（外部依存なし）
 * asChild=true のとき、子要素をそのまま使い className/props を合成して返す
 */
function SimpleSlot(
  props: React.HTMLAttributes<HTMLElement> & { children?: React.ReactNode },
) {
  const { children, ...rest } = props;
  if (React.isValidElement(children)) {
    const prev = (children.props as { className?: string })?.className;
    return React.cloneElement(children as any, {
      ...rest,
      className: cn(prev, (rest as any).className),
    });
  }
  // 子が要素でない場合は fallback ボタン
  return <button {...(rest as any)}>{children}</button>;
}

/** cva っぽい超軽量バリアント合成（外部依存なし） */
type BtnVariant =
  | "default"
  | "destructive"
  | "outline"
  | "secondary"
  | "ghost"
  | "link"
  /** ◀ 追加：濃いソリッド（期待値の「＋」ボタン用） */
  | "solid";

type BtnSize = "default" | "sm" | "lg" | "icon";

function buttonVariants(opts?: {
  variant?: BtnVariant;
  size?: BtnSize;
  className?: string;
}) {
  const { variant = "default", size = "default", className } = opts ?? {};

  const base =
    "inline-flex items-center justify-center gap-2 whitespace-nowrap text-sm font-medium " +
    "transition-all disabled:pointer-events-none disabled:opacity-50 " +
    // svg のサイズ・クリック無効
    "[&_svg]:pointer-events-none [&_svg:not([class*='size-'])]:size-4 shrink-0 [&_svg]:shrink-0 " +
    // フォーカスリング
    "outline-none focus-visible:border-[hsl(var(--ring))] focus-visible:ring-[hsl(var(--ring))/0.5] focus-visible:ring-[3px] " +
    "aria-invalid:ring-[hsl(var(--destructive))/0.2] dark:aria-invalid:ring-[hsl(var(--destructive))/0.4] " +
    "aria-invalid:border-[hsl(var(--destructive))]";

  const byVariant: Record<BtnVariant, string> = {
    default:
      "rounded-md bg-[hsl(var(--primary))] text-[hsl(var(--primary-foreground))] hover:bg-[hsl(var(--primary))/0.9] border border-transparent",
    destructive:
      "rounded-md bg-[hsl(var(--destructive))] text-white hover:bg-[hsl(var(--destructive))/0.9] border border-transparent",
    outline:
      "rounded-md border bg-[hsl(var(--background))] text-[hsl(var(--foreground))] " +
      "hover:bg-[hsl(var(--accent))] hover:text-[hsl(var(--accent-foreground))]",
    secondary:
      "rounded-md bg-[hsl(var(--secondary))] text-[hsl(var(--secondary-foreground))] hover:bg-[hsl(var(--secondary))/0.8] border border-transparent",
    ghost:
      "rounded-md hover:bg-[hsl(var(--accent))] hover:text-[hsl(var(--accent-foreground))] border border-transparent",
    link: "rounded-none text-[hsl(var(--primary))] underline-offset-4 hover:underline border border-transparent",
    /** 期待値の黒に近い濃紺ソリッド。角丸はやや大きめ、文字/アイコンは白 */
    solid:
      "rounded-lg bg-[#0a0a1b] text-white hover:opacity-95 border border-transparent shadow-[0_2px_8px_rgba(2,6,23,0.10)]",
  };

  const bySize: Record<BtnSize, string> = {
    default: "h-9 px-4 py-2 has-[>svg]:px-3",
    sm: "h-8 rounded-md gap-1.5 px-3 has-[>svg]:px-2.5",
    lg: "h-10 rounded-md px-6 has-[>svg]:px-4",
    // 角丸スクエア 36px。見た目を揃えるため solid と相性の良い rounded-lg を強制
    icon: "size-9 rounded-lg p-0",
  };

  return cn(base, byVariant[variant], bySize[size], className);
}

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  asChild?: boolean;
  variant?: BtnVariant;
  size?: BtnSize;
}

/**
 * 外部ライブラリ不使用の Button
 * - asChild: true で子要素をラップせずに描画（SimpleSlot）
 * - variant/size は自前の合成器（buttonVariants）で生成
 */
export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, children, ...props }, ref) => {
    const classes = buttonVariants({ variant, size, className });

    if (asChild) {
      // 子に className/属性を移譲（ref は移譲できない要素もある点に注意）
      return (
        <SimpleSlot className={classes} {...(props as any)}>
          {children}
        </SimpleSlot>
      );
    }

    return (
      <button data-slot="button" ref={ref} className={classes} {...props}>
        {children}
      </button>
    );
  },
);
Button.displayName = "Button";

export { buttonVariants };
