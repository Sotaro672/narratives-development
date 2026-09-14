// frontend/mall/src/features/howToUse/presentation/components/common/HowToUseSection.tsx

import type { ReactNode } from "react";

type HowToUseSectionProps = {
  title: string;
  children: ReactNode;
  id?: string;
};

export default function HowToUseSection({
  title,
  children,
  id,
}: HowToUseSectionProps) {
  return (
    <section
      id={id}
      className="how-to-use-article-section"
    >
      <h2 className="how-to-use-article-section__title">
        {title}
      </h2>

      <div className="how-to-use-article-section__content">
        {children}
      </div>
    </section>
  );
}