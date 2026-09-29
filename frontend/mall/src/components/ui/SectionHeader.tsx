// frontend/mall/src/components/ui/SectionHeader.tsx

import type { ReactNode } from "react";

import "./sectionHeader.css";

type SectionHeaderTitleTag = "h1" | "h2" | "h3";
export type SectionHeaderTitleSize = "default" | "sm" | "md";

type SectionHeaderProps = {
  title?: ReactNode;
  titleAs?: SectionHeaderTitleTag;
  titleSize?: SectionHeaderTitleSize;
  eyebrow?: ReactNode;
  right?: ReactNode;
  children?: ReactNode;
  className?: string;
};

export default function SectionHeader({
  title,
  titleAs: TitleTag = "h1",
  titleSize = "default",
  eyebrow,
  right,
  children,
  className,
}: SectionHeaderProps) {
  const classes = [
    "ui-section-header",
    titleSize !== "default" ? `ui-section-header--title-${titleSize}` : "",
    className,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <div className={classes}>
      <div className="ui-section-header__body">
        {eyebrow ? (
          <p className="ui-section-header__eyebrow">{eyebrow}</p>
        ) : null}

        {title ? (
          <TitleTag className="ui-section-header__title">
            {title}
          </TitleTag>
        ) : null}

        {children}
      </div>

      {right ? (
        <div className="ui-section-header__right">
          {right}
        </div>
      ) : null}
    </div>
  );
}