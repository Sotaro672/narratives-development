// frontend/mall/src/features/howToUse/presentation/components/common/HowToUseArticle.tsx

import type { ReactNode } from "react";

type HowToUseArticleProps = {
  description: string;
  children: ReactNode;
};

export default function HowToUseArticle({
  description,
  children,
}: HowToUseArticleProps) {
  return (
    <article className="how-to-use-article">
      <p className="how-to-use-detail-page__description">
        {description}
      </p>

      <div className="how-to-use-article__content">
        {children}
      </div>
    </article>
  );
}