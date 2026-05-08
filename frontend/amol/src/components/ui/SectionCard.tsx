// frontend/amol/src/components/ui/SectionCard.tsx
import type { ReactNode } from "react";

import "./section-card.css";

type SectionCardProps = {
  children: ReactNode;
  className?: string;
};

export default function SectionCard(props: SectionCardProps) {
  const { children, className } = props;

  return (
    <section className={["ui-section-card", className].filter(Boolean).join(" ")}>
      {children}
    </section>
  );
}