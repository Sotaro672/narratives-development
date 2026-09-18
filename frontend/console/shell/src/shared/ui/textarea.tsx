// frontend/console/shell/src/shared/ui/textarea.tsx

import * as React from "react";

import "./textarea.css";

function cn(...classes: Array<string | undefined | null | false>) {
  return classes.filter(Boolean).join(" ");
}

export type TextareaSize = "default" | "medium" | "large";

export type TextareaProps =
  React.TextareaHTMLAttributes<HTMLTextAreaElement> & {
    containerClassName?: string;
    size?: TextareaSize;
  };

const sizeClassNames: Record<TextareaSize, string> = {
  default: "textarea--default",
  medium: "textarea--medium",
  large: "textarea--large",
};

export const Textarea = React.forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ className, containerClassName, size = "default", ...props }, ref) => {
    return (
      <div className={containerClassName}>
        <textarea
          ref={ref}
          className={cn("textarea", sizeClassNames[size], className)}
          {...props}
        />
      </div>
    );
  },
);

Textarea.displayName = "Textarea";

export default Textarea;