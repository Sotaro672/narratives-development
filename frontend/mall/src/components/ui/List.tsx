// frontend/mall/src/components/ui/List.tsx

import type { HTMLAttributes, ReactNode } from "react";
import { ChevronRight } from "lucide-react";

import "./List.css";

type ListProps = HTMLAttributes<HTMLDivElement> & {
  children: ReactNode;
};

type ListItemProps = {
  label: string;
  onClick: () => void | Promise<void>;
  danger?: boolean;
  disabled?: boolean;
  right?: ReactNode;
  className?: string;
};

export function ListItem({
  label,
  onClick,
  danger = false,
  disabled = false,
  right,
  className = "",
}: ListItemProps) {
  const classes = [
    "ui-list__row",
    danger ? "ui-list__row--danger" : "",
    disabled ? "ui-list__row--disabled" : "",
    className,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <div className={classes}>
      <button
        type="button"
        className="ui-list__action"
        onClick={onClick}
        disabled={disabled}
      >
        <span className="ui-list__label">{label}</span>

        <span className="ui-list__right" aria-hidden="true">
          {right ?? <ChevronRight size={20} strokeWidth={2} />}
        </span>
      </button>
    </div>
  );
}

export default function List({
  children,
  className = "",
  ...props
}: ListProps) {
  const classes = ["ui-list", className].filter(Boolean).join(" ");

  return (
    <div className={classes} {...props}>
      {children}
    </div>
  );
}