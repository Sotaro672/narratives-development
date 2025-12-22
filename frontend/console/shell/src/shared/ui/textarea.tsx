// frontend/console/shell/src/shared/ui/textarea.tsx

import * as React from "react";

function cn(...classes: Array<string | undefined | null | false>) {
  return classes.filter(Boolean).join(" ");
}

export type TextareaProps =
  React.TextareaHTMLAttributes<HTMLTextAreaElement> & {
    containerClassName?: string;
  };

export const Textarea = React.forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ className, containerClassName, ...props }, ref) => {
    return (
      <div className={containerClassName}>
        <textarea
          ref={ref}
          className={cn(
            // base
            "flex min-h-[96px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm",
            // focus
            "focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring",
            // disabled
            "disabled:cursor-not-allowed disabled:opacity-50",
            // placeholder
            "placeholder:text-muted-foreground",
            className,
          )}
          {...props}
        />
      </div>
    );
  },
);

Textarea.displayName = "Textarea";

export default Textarea;
