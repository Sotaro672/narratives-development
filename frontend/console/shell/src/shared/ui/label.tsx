// frontend/console/shell/src/shared/ui/label.tsx

import * as React from "react";

import "./label.css";

function cn(...classes: Array<string | undefined | false | null>): string {
  return classes.filter(Boolean).join(" ");
}

export interface LabelProps
  extends React.LabelHTMLAttributes<HTMLLabelElement> {}

export const Label = React.forwardRef<HTMLLabelElement, LabelProps>(
  ({ className, ...props }, ref) => (
    <label
      ref={ref}
      className={cn("label", className)}
      {...props}
    />
  ),
);

Label.displayName = "Label";