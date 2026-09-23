// frontend/console/shell/src/shared/ui/avatarIcon.tsx

import * as React from "react";

import "./avatarIcon.css";

function cn(...classes: Array<string | undefined | false | null>): string {
  return classes.filter(Boolean).join(" ");
}

export type AvatarIconProps = Omit<
  React.ImgHTMLAttributes<HTMLImageElement>,
  "src"
> & {
  src?: string | null;
};

export const AvatarIcon = React.forwardRef<
  HTMLImageElement,
  AvatarIconProps
>(
  (
    {
      src,
      alt = "",
      className,
      ...props
    },
    ref,
  ) => {
    if (!src) {
      return null;
    }

    return (
      <img
        ref={ref}
        src={src}
        alt={alt}
        className={cn("avatar-icon", className)}
        {...props}
      />
    );
  },
);

AvatarIcon.displayName = "AvatarIcon";

export default AvatarIcon;