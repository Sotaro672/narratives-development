// frontend/amol/src/components/ui/SectionHeader.tsx
import type { ReactNode } from "react";

import "./section-header.css";

type SectionHeaderTitleTag = "h1" | "h2" | "h3";

type SectionHeaderProps = {
  title?: ReactNode;
  titleAs?: SectionHeaderTitleTag;
  eyebrow?: ReactNode;
  right?: ReactNode;
  children?: ReactNode;
  className?: string;
};

export default function SectionHeader(props: SectionHeaderProps) {
  const {
    title,
    titleAs: TitleTag = "h1",
    eyebrow,
    right,
    children,
    className,
  } = props;

  return (
    <div className={["ui-section-header", className].filter(Boolean).join(" ")}>
      <div className="ui-section-header__body">
        {eyebrow ? (
          <p className="ui-section-header__eyebrow">{eyebrow}</p>
        ) : null}

        {title ? (
          <TitleTag className="ui-section-header__title">{title}</TitleTag>
        ) : null}

        {children}
      </div>

      {right ? <div className="ui-section-header__right">{right}</div> : null}
    </div>
  );
}