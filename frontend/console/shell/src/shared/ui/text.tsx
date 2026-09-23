// frontend/console/shell/src/shared/ui/text.tsx

import * as React from "react";

import "./text.css";

function cn(...classes: Array<string | undefined | false | null>): string {
  return classes.filter(Boolean).join(" ");
}

export type TextSize =
  | "xs"
  | "sm"
  | "md";

export type TextTone =
  | "default"
  | "muted"
  | "destructive";

export type TextWeight =
  | "normal"
  | "medium"
  | "semibold"
  | "bold";

export type TextWrap =
  | "normal"
  | "pre-wrap"
  | "anywhere"
  | "pre-wrap-anywhere"
  | "nowrap";

export type TextElement =
  | "span"
  | "p"
  | "div";

export type TextProps = React.HTMLAttributes<HTMLElement> & {
  as?: TextElement;
  size?: TextSize;
  tone?: TextTone;
  weight?: TextWeight;
  wrap?: TextWrap;
};

const sizeClassNames: Record<TextSize, string> = {
  xs: "text--xs",
  sm: "text--sm",
  md: "text--md",
};

const toneClassNames: Record<TextTone, string> = {
  default: "text--default",
  muted: "text--muted",
  destructive: "text--destructive",
};

const weightClassNames: Record<TextWeight, string> = {
  normal: "text--weight-normal",
  medium: "text--weight-medium",
  semibold: "text--weight-semibold",
  bold: "text--weight-bold",
};

const wrapClassNames: Record<TextWrap, string> = {
  normal: "text--wrap-normal",
  "pre-wrap": "text--wrap-pre-wrap",
  anywhere: "text--wrap-anywhere",
  "pre-wrap-anywhere": "text--wrap-pre-wrap-anywhere",
  nowrap: "text--wrap-nowrap",
};

export const Text = React.forwardRef<HTMLElement, TextProps>(
  (
    {
      as: Component = "span",
      className,
      size = "sm",
      tone = "default",
      weight = "normal",
      wrap = "normal",
      ...props
    },
    ref,
  ) => (
    <Component
      ref={ref as React.Ref<
        HTMLSpanElement & HTMLParagraphElement & HTMLDivElement
      >}
      className={cn(
        "text",
        sizeClassNames[size],
        toneClassNames[tone],
        weightClassNames[weight],
        wrapClassNames[wrap],
        className,
      )}
      {...props}
    />
  ),
);

Text.displayName = "Text";

export default Text;