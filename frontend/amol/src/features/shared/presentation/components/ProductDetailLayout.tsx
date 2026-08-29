// frontend\amol\src\features\shared\presentation\components\ProductDetailLayout.tsx

import type { ReactNode } from "react";

export type ProductDetailLayoutProps = {
  media: ReactNode;
  mediaFooter?: ReactNode;
  children: ReactNode;
  className?: string;
  mediaColumnClassName?: string;
  contentClassName?: string;
};

function joinClassNames(...classNames: Array<string | undefined | false>): string {
  return classNames.filter(Boolean).join(" ");
}

export default function ProductDetailLayout({
  media,
  mediaFooter,
  children,
  className,
  mediaColumnClassName,
  contentClassName,
}: ProductDetailLayoutProps) {
  return (
    <section className={joinClassNames("product-detail__layout", className)}>
      <div className={joinClassNames("product-detail__media-column", mediaColumnClassName)}>
        {media}

        {mediaFooter ? (
          <div className="product-detail__media-footer">
            {mediaFooter}
          </div>
        ) : null}
      </div>

      <div className={joinClassNames("product-detail__content", contentClassName)}>
        {children}
      </div>
    </section>
  );
}