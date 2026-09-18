// frontend/console/shell/src/shared/ui/card.tsx

import * as React from "react";

import "./card.css";

function cn(...classes: Array<string | undefined | false | null>): string {
  return classes.filter(Boolean).join(" ");
}

export type CardProps = React.HTMLAttributes<HTMLDivElement> & {
  elevated?: boolean;
  largeRadius?: boolean;
};

export const Card = React.forwardRef<HTMLDivElement, CardProps>(
  (
    {
      className,
      elevated = false,
      largeRadius = false,
      ...props
    },
    ref,
  ) => (
    <div
      ref={ref}
      className={cn(
        "card",
        elevated && "card--elevated",
        largeRadius && "card--large-radius",
        className,
      )}
      {...props}
    />
  ),
);
Card.displayName = "Card";

export type CardHeaderProps = React.HTMLAttributes<HTMLDivElement>;

export const CardHeader = React.forwardRef<HTMLDivElement, CardHeaderProps>(
  ({ className, ...props }, ref) => (
    <div
      ref={ref}
      className={cn("card__header", className)}
      {...props}
    />
  ),
);
CardHeader.displayName = "CardHeader";

export type CardHeaderLeftProps = React.HTMLAttributes<HTMLDivElement>;

export const CardHeaderLeft = React.forwardRef<HTMLDivElement, CardHeaderLeftProps>(
  ({ className, ...props }, ref) => (
    <div
      ref={ref}
      className={cn("card__header-left", className)}
      {...props}
    />
  ),
);
CardHeaderLeft.displayName = "CardHeaderLeft";

export type CardHeaderIconProps = React.HTMLAttributes<HTMLDivElement> & {
  variant?: "default" | "primary";
};

export const CardHeaderIcon = React.forwardRef<HTMLDivElement, CardHeaderIconProps>(
  (
    {
      className,
      variant = "default",
      ...props
    },
    ref,
  ) => (
    <div
      ref={ref}
      className={cn(
        "card__header-icon",
        variant === "primary" && "card__header-icon--primary",
        className,
      )}
      {...props}
    />
  ),
);
CardHeaderIcon.displayName = "CardHeaderIcon";

export type CardTitleProps = React.HTMLAttributes<HTMLHeadingElement> & {
  strong?: boolean;
  truncate?: boolean;
};

export const CardTitle = React.forwardRef<HTMLHeadingElement, CardTitleProps>(
  (
    {
      className,
      strong = false,
      truncate = false,
      ...props
    },
    ref,
  ) => (
    <h3
      ref={ref}
      className={cn(
        "card__title",
        strong && "card__title--strong",
        truncate && "card__title--truncate",
        className,
      )}
      {...props}
    />
  ),
);
CardTitle.displayName = "CardTitle";

export type CardBadgeProps = React.HTMLAttributes<HTMLSpanElement> & {
  variant?: "default" | "primary";
};

export const CardBadge = React.forwardRef<HTMLSpanElement, CardBadgeProps>(
  (
    {
      className,
      variant = "default",
      ...props
    },
    ref,
  ) => (
    <span
      ref={ref}
      className={cn(
        "card__badge",
        variant === "primary" && "card__badge--primary",
        className,
      )}
      {...props}
    />
  ),
);
CardBadge.displayName = "CardBadge";

export type CardContentProps = React.HTMLAttributes<HTMLDivElement> & {
  size?: "default" | "large";
};

export const CardContent = React.forwardRef<HTMLDivElement, CardContentProps>(
  (
    {
      className,
      size = "default",
      ...props
    },
    ref,
  ) => (
    <div
      ref={ref}
      className={cn(
        "card__content",
        size === "large" && "card__content--large",
        className,
      )}
      {...props}
    />
  ),
);
CardContent.displayName = "CardContent";

export type CardFooterProps = React.HTMLAttributes<HTMLDivElement>;

export const CardFooter = React.forwardRef<HTMLDivElement, CardFooterProps>(
  ({ className, ...props }, ref) => (
    <div
      ref={ref}
      className={cn("card__footer", className)}
      {...props}
    />
  ),
);
CardFooter.displayName = "CardFooter";

export type CardLabelProps = React.LabelHTMLAttributes<HTMLLabelElement> & {
  strong?: boolean;
};

export const CardLabel = React.forwardRef<HTMLLabelElement, CardLabelProps>(
  (
    {
      className,
      strong = false,
      ...props
    },
    ref,
  ) => (
    <label
      ref={ref}
      className={cn(
        "card__label",
        strong && "card__label--strong",
        className,
      )}
      {...props}
    />
  ),
);
CardLabel.displayName = "CardLabel";

export type CardInputProps = React.InputHTMLAttributes<HTMLInputElement> & {
  sizeVariant?: "default" | "large";
};

export const CardInput = React.forwardRef<HTMLInputElement, CardInputProps>(
  (
    {
      className,
      sizeVariant = "default",
      ...props
    },
    ref,
  ) => (
    <input
      ref={ref}
      className={cn(
        "card__input",
        sizeVariant === "large" && "card__input--large",
        className,
      )}
      {...props}
    />
  ),
);
CardInput.displayName = "CardInput";

export type CardSelectProps = React.SelectHTMLAttributes<HTMLSelectElement> & {
  sizeVariant?: "default" | "large";
};

export const CardSelect = React.forwardRef<HTMLSelectElement, CardSelectProps>(
  (
    {
      className,
      sizeVariant = "default",
      ...props
    },
    ref,
  ) => (
    <select
      ref={ref}
      className={cn(
        "card__select",
        sizeVariant === "large" && "card__select--large",
        className,
      )}
      {...props}
    />
  ),
);
CardSelect.displayName = "CardSelect";

export type CardSelectWrapProps = React.HTMLAttributes<HTMLDivElement>;

export const CardSelectWrap = React.forwardRef<HTMLDivElement, CardSelectWrapProps>(
  ({ className, ...props }, ref) => (
    <div
      ref={ref}
      className={cn("card__select-wrap", className)}
      {...props}
    />
  ),
);
CardSelectWrap.displayName = "CardSelectWrap";

export type CardReadonlyProps = React.HTMLAttributes<HTMLDivElement> & {
  inputLike?: boolean;
  size?: "default" | "large";
};

export const CardReadonly = React.forwardRef<HTMLDivElement, CardReadonlyProps>(
  (
    {
      className,
      inputLike = false,
      size = "default",
      ...props
    },
    ref,
  ) => (
    <div
      ref={ref}
      className={cn(
        "card__readonly",
        inputLike && "card__readonly--input",
        size === "large" && "card__readonly--large",
        className,
      )}
      {...props}
    />
  ),
);
CardReadonly.displayName = "CardReadonly";

export type CardViewValueProps = React.HTMLAttributes<HTMLDivElement>;

export const CardViewValue = React.forwardRef<HTMLDivElement, CardViewValueProps>(
  ({ className, ...props }, ref) => (
    <div
      ref={ref}
      className={cn("card__view-value", className)}
      {...props}
    />
  ),
);
CardViewValue.displayName = "CardViewValue";

export type CardTextareaProps = React.TextareaHTMLAttributes<HTMLTextAreaElement>;

export const CardTextarea = React.forwardRef<HTMLTextAreaElement, CardTextareaProps>(
  ({ className, ...props }, ref) => (
    <textarea
      ref={ref}
      className={cn("card__textarea", className)}
      {...props}
    />
  ),
);
CardTextarea.displayName = "CardTextarea";

export type CardSuffixProps = React.HTMLAttributes<HTMLDivElement>;

export const CardSuffix = React.forwardRef<HTMLDivElement, CardSuffixProps>(
  ({ className, ...props }, ref) => (
    <div
      ref={ref}
      className={cn("card__suffix", className)}
      {...props}
    />
  ),
);
CardSuffix.displayName = "CardSuffix";

export type CardChipsProps = React.HTMLAttributes<HTMLDivElement>;

export const CardChips = React.forwardRef<HTMLDivElement, CardChipsProps>(
  ({ className, ...props }, ref) => (
    <div
      ref={ref}
      className={cn("card__chips", className)}
      {...props}
    />
  ),
);
CardChips.displayName = "CardChips";

export type CardChipProps = React.HTMLAttributes<HTMLDivElement>;

export const CardChip = React.forwardRef<HTMLDivElement, CardChipProps>(
  ({ className, ...props }, ref) => (
    <div
      ref={ref}
      className={cn("card__chip", className)}
      {...props}
    />
  ),
);
CardChip.displayName = "CardChip";

export type CardButtonProps = React.ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: "default" | "primary" | "secondary" | "danger";
};

export const CardButton = React.forwardRef<HTMLButtonElement, CardButtonProps>(
  (
    {
      className,
      variant = "default",
      type = "button",
      ...props
    },
    ref,
  ) => (
    <button
      ref={ref}
      type={type}
      className={cn(
        "card__button",
        variant !== "default" && `card__button--${variant}`,
        className,
      )}
      {...props}
    />
  ),
);
CardButton.displayName = "CardButton";

export type CardFieldProps = React.HTMLAttributes<HTMLDivElement> & {
  full?: boolean;
};

export const CardField = React.forwardRef<HTMLDivElement, CardFieldProps>(
  (
    {
      className,
      full = false,
      ...props
    },
    ref,
  ) => (
    <div
      ref={ref}
      className={cn(
        "card__field",
        full && "card__field--full",
        className,
      )}
      {...props}
    />
  ),
);
CardField.displayName = "CardField";

export type CardFieldsProps = React.HTMLAttributes<HTMLDivElement>;

export const CardFields = React.forwardRef<HTMLDivElement, CardFieldsProps>(
  ({ className, ...props }, ref) => (
    <div
      ref={ref}
      className={cn("card__fields", className)}
      {...props}
    />
  ),
);
CardFields.displayName = "CardFields";