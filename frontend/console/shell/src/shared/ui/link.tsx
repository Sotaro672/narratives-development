// frontend/console/shell/src/shared/ui/link.tsx

import * as React from "react";

import "./link.css";

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
      className: cn(previousClassName, rest.className),
    });
  }

  return null;
}

type LinkCommonProps = {
  asChild?: boolean;
};

export type LinkProps =
  | (
      React.ButtonHTMLAttributes<HTMLButtonElement> &
      LinkCommonProps & {
        as?: "button";
      }
    )
  | (
      React.AnchorHTMLAttributes<HTMLAnchorElement> &
      LinkCommonProps & {
        as: "a";
      }
    );

export const Link = React.forwardRef<
  HTMLButtonElement | HTMLAnchorElement,
  LinkProps
>(
  (
    {
      as = "button",
      asChild = false,
      className,
      children,
      ...props
    },
    ref,
  ) => {
    const classes = cn("link", className);

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

    if (as === "a") {
      return (
        <a
          data-slot="link"
          ref={ref as React.Ref<HTMLAnchorElement>}
          className={classes}
          {...(props as React.AnchorHTMLAttributes<HTMLAnchorElement>)}
        >
          {children}
        </a>
      );
    }

    const buttonProps =
      props as React.ButtonHTMLAttributes<HTMLButtonElement>;

    return (
      <button
        data-slot="link"
        ref={ref as React.Ref<HTMLButtonElement>}
        {...buttonProps}
        type={buttonProps.type ?? "button"}
        className={classes}
      >
        {children}
      </button>
    );
  },
);

Link.displayName = "Link";

export default Link;