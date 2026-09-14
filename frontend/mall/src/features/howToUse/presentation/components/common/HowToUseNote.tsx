// frontend/mall/src/features/howToUse/presentation/components/common/HowToUseNote.tsx

import type { ReactNode } from "react";

type HowToUseNoteProps = {
  children: ReactNode;
  title?: string;
};

export default function HowToUseNote({
  children,
  title,
}: HowToUseNoteProps) {
  return (
    <aside className="how-to-use-note">
      {title ? (
        <p className="how-to-use-note__title">
          {title}
        </p>
      ) : null}

      <div className="how-to-use-note__content">
        {children}
      </div>
    </aside>
  );
}