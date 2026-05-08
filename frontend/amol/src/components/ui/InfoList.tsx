// frontend/amol/src/components/ui/InfoList.tsx
import type { ReactNode } from "react";

import "./info-list.css";

export type InfoListRow = {
  label: ReactNode;
  value: ReactNode;
  key?: string;
};

type InfoListProps = {
  rows?: InfoListRow[];
  children?: ReactNode;
  className?: string;
};

type InfoRowProps = {
  label: ReactNode;
  children: ReactNode;
  className?: string;
};

export function InfoRow(props: InfoRowProps) {
  const { label, children, className } = props;

  return (
    <div className={["ui-info-row", className].filter(Boolean).join(" ")}>
      <span className="ui-info-row__label">{label}</span>
      <strong className="ui-info-row__value">{children}</strong>
    </div>
  );
}

export default function InfoList(props: InfoListProps) {
  const { rows, children, className } = props;

  return (
    <div className={["ui-info-list", className].filter(Boolean).join(" ")}>
      {rows?.map((row, index) => (
        <InfoRow label={row.label} key={row.key ?? String(index)}>
          {row.value}
        </InfoRow>
      ))}

      {children}
    </div>
  );
}