// frontend/console/shell/src/shared/ui/color.tsx

import type {
  HTMLAttributes,
  ReactNode,
} from "react";

import "./color.css";

export type ColorSwatchSize =
  | "sm"
  | "md";

export type ColorSwatchShape =
  | "circle"
  | "square";

export type ColorSwatchProps = {
  color?: string | null;
  size?: ColorSwatchSize;
  shape?: ColorSwatchShape;
  className?: string;
  title?: string;
  ariaLabel?: string;
};

export type ColorValueProps =
  Omit<
    HTMLAttributes<HTMLSpanElement>,
    "children" | "color"
  > & {
    color?: string | null;
    children: ReactNode;
    size?: ColorSwatchSize;
    shape?: ColorSwatchShape;
    swatchClassName?: string;
    swatchTitle?: string;
    swatchAriaLabel?: string;
  };

function classNames(
  ...values: Array<string | undefined | false | null>
): string {
  return values.filter(Boolean).join(" ");
}

function normalizeColor(
  color?: string | null,
): string | undefined {
  const normalized = String(color ?? "").trim();

  return normalized || undefined;
}

export function ColorSwatch({
  color,
  size = "sm",
  shape = "circle",
  className,
  title,
  ariaLabel,
}: ColorSwatchProps) {
  const normalizedColor = normalizeColor(color);

  return (
    <span
      className={classNames(
        "ui-color__swatch",
        `ui-color__swatch--${size}`,
        `ui-color__swatch--${shape}`,
        className,
      )}
      style={
        normalizedColor
          ? { backgroundColor: normalizedColor }
          : undefined
      }
      title={title}
      role={ariaLabel ? "img" : undefined}
      aria-label={ariaLabel}
      aria-hidden={ariaLabel ? undefined : true}
    />
  );
}

export function ColorValue({
  color,
  children,
  size = "sm",
  shape = "circle",
  className,
  swatchClassName,
  swatchTitle,
  swatchAriaLabel,
  ...props
}: ColorValueProps) {
  const normalizedColor = normalizeColor(color);

  return (
    <span
      {...props}
      className={classNames(
        "ui-color",
        className,
      )}
    >
      {normalizedColor ? (
        <ColorSwatch
          color={normalizedColor}
          size={size}
          shape={shape}
          className={swatchClassName}
          title={swatchTitle}
          ariaLabel={swatchAriaLabel}
        />
      ) : null}

      <span className="ui-color__label">
        {children}
      </span>
    </span>
  );
}