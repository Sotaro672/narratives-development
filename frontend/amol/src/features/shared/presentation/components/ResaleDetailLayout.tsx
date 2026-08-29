// frontend/amol/src/features/shared/presentation/componentns/ResaleDetailLayout.tsx

import type { ReactNode } from "react";

export type ResaleDetailLayoutProps = {
  media: ReactNode;
  mediaFooter?: ReactNode;
  children: ReactNode;
  className?: string;
  mediaColumnClassName?: string;
  contentClassName?: string;
};

function joinClassNames(
  ...classNames: Array<string | undefined | false>
): string {
  return classNames.filter(Boolean).join(" ");
}

export default function ResaleDetailLayout({
  media,
  mediaFooter,
  children,
  className,
  mediaColumnClassName,
  contentClassName,
}: ResaleDetailLayoutProps) {
  return (
    <section
      className={joinClassNames(
        "resale-product-detail__layout",
        className,
      )}
    >
      <div
        className={joinClassNames(
          "resale-product-detail__media-column",
          mediaColumnClassName,
        )}
      >
        {media}

        {mediaFooter ? (
          <div className="resale-product-detail__media-footer">
            {mediaFooter}
          </div>
        ) : null}
      </div>

      <div
        className={joinClassNames(
          "resale-product-detail__content",
          contentClassName,
        )}
      >
        {children}
      </div>
    </section>
  );
}