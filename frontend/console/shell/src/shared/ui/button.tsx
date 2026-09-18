// frontend/console/shell/src/shared/ui/button.tsx

import * as React from "react";

import "./button.css";

function cn(...classes: Array<string | undefined | false | null>): string {
  return classes.filter(Boolean).join(" ");
}

function SimpleSlot(
  props: React.HTMLAttributes<HTMLElement> & { children?: React.ReactNode },
) {
  const { children, ...rest } = props;

  if (React.isValidElement(children)) {
    const previousClassName = (
      children.props as {
        className?: string;
      }
    )?.className;

    return React.cloneElement(children as React.ReactElement<any>, {
      ...rest,
      className: cn(
        previousClassName,
        rest.className,
      ),
    });
  }

  return (
    <button {...(rest as React.ButtonHTMLAttributes<HTMLButtonElement>)}>
      {children}
    </button>
  );
}

export type BtnVariant =
  | "default"
  | "destructive"
  | "outline"
  | "secondary"
  | "ghost"
  | "link"
  | "solid";

export type BtnSize =
  | "default"
  | "sm"
  | "lg"
  | "icon";

export function buttonVariants(opts?: {
  variant?: BtnVariant;
  size?: BtnSize;
  className?: string;
}): string {
  const {
    variant = "default",
    size = "default",
    className,
  } = opts ?? {};

  return cn(
    "button",
    `button--${variant}`,
    `button--${size}`,
    className,
  );
}

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  asChild?: boolean;
  variant?: BtnVariant;
  size?: BtnSize;
}

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  (
    {
      className,
      variant = "default",
      size = "default",
      asChild = false,
      children,
      type = "button",
      ...props
    },
    ref,
  ) => {
    const classes = buttonVariants({
      variant,
      size,
      className,
    });

    if (asChild) {
      return (
        <SimpleSlot
          className={classes}
          {...(props as React.HTMLAttributes<HTMLElement>)}
        >
          {children}
        </SimpleSlot>
      );
    }

    return (
      <button
        data-slot="button"
        ref={ref}
        type={type}
        className={classes}
        {...props}
      >
        {children}
      </button>
    );
  },
);

Button.displayName = "Button";