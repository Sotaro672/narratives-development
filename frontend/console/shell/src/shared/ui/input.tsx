// frontend/shared/ui/input.tsx
import * as React from "react";

/** className 結合ヘルパー */
function cn(...classes: Array<string | undefined | false | null>) {
  return classes.filter(Boolean).join(" ");
}

/**
 * Input
 * - productBlueprintCard.css の .pbc .input / .pbc .readonly にフックするため
 *   既定で "input" クラスを付与。readonly では "readonly" を追加。
 * - 他画面でも使えるように追加 className で上書き可能。
 */
export interface InputProps
  extends React.InputHTMLAttributes<HTMLInputElement> {
  variant?: "default" | "readonly";
}

export const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className, variant = "default", readOnly, ...props }, ref) => {
    const classes = cn(
      "input",
      variant === "readonly" && "readonly",
      className
    );

    return (
      <input
        ref={ref}
        className={classes}
        readOnly={variant === "readonly" ? true : readOnly}
        {...props}
      />
    );
  }
);
Input.displayName = "Input";
