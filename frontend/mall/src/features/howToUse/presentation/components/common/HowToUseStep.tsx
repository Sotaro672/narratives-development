// frontend/mall/src/features/howToUse/presentation/components/common/HowToUseStep.tsx

import type { ReactNode } from "react";

type HowToUseStepProps = {
  children: ReactNode;
};

export default function HowToUseStep({
  children,
}: HowToUseStepProps) {
  return (
    <li className="how-to-use-step">
      <div className="how-to-use-step__content">
        {children}
      </div>
    </li>
  );
}