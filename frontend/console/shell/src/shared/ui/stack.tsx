// frontend/console/shell/src/shared/ui/stack.tsx

import * as React from "react";

import "./stack.css";

function cn(...classes: Array<string | undefined | false | null>): string {
  return classes.filter(Boolean).join(" ");
}

export type StackGap =
  | "xs"
  | "sm"
  | "md"
  | "lg";

export type StackProps = React.HTMLAttributes<HTMLDivElement> & {
  gap?: StackGap;
};

export const Stack = React.forwardRef<HTMLDivElement, StackProps>(
  (
    {
      gap = "sm",
      className,
      ...props
    },
    ref,
  ) => (
    <div
      ref={ref}
      className={cn(
        "stack",
        `stack--gap-${gap}`,
        className,
      )}
      {...props}
    />
  ),
);

Stack.displayName = "Stack";

export default Stack;