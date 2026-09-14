// frontend/mall/src/features/howToUse/presentation/components/common/HowToUseStepList.tsx

import type { ReactNode } from "react";

type HowToUseStepListProps = {
  children: ReactNode;
};

export default function HowToUseStepList({
  children,
}: HowToUseStepListProps) {
  return (
    <ol className="how-to-use-step-list">
      {children}
    </ol>
  );
}