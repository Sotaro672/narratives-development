// frontend/mall/src/components/ui/List.tsx

import type {
  HTMLAttributes,
  KeyboardEvent,
  ReactNode,
} from "react";
import { ChevronRight } from "lucide-react";

import DateDisplay from "./Date";
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

type ListRowProps = {
  leading?: ReactNode;
  title: ReactNode;
  subLabel?: ReactNode;
  dateValue?: string | null;
  preview?: ReactNode;
  meta?: ReactNode;
  attention?: boolean;
  selected?: boolean;
  busy?: boolean;
  disabled?: boolean;
  ariaLabel: string;
  onClick: () => void | Promise<void>;
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

export function ListRow({
  leading,
  title,
  subLabel,
  dateValue,
  preview,
  meta,
  attention = false,
  selected,
  busy = false,
  disabled = false,
  ariaLabel,
  onClick,
  className = "",
}: ListRowProps) {
  const unavailable = busy || disabled;

  const classes = [
    "ui-list__row",
    "ui-list__row--rich",
    attention ? "ui-list__row--attention" : "",
    selected === true ? "ui-list__row--selected" : "",
    unavailable ? "ui-list__row--disabled" : "",
    className,
  ]
    .filter(Boolean)
    .join(" ");

  const handleOpen = () => {
    if (unavailable) {
      return;
    }

    void onClick();
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLElement>) => {
    if (
      unavailable ||
      (event.key !== "Enter" && event.key !== " ")
    ) {
      return;
    }

    event.preventDefault();
    void onClick();
  };

  return (
    <article
      className={classes}
      role="button"
      tabIndex={unavailable ? -1 : 0}
      aria-label={ariaLabel}
      aria-pressed={
        selected === undefined ? undefined : selected
      }
      aria-busy={busy || undefined}
      aria-disabled={unavailable || undefined}
      onClick={handleOpen}
      onKeyDown={handleKeyDown}
    >
      {leading ? (
        <div className="ui-list__leading" aria-hidden="true">
          {leading}
        </div>
      ) : null}

      <div className="ui-list__body">
        <div className="ui-list__head">
          <div className="ui-list__title-wrap">
            <h2 className="ui-list__title">{title}</h2>

            {subLabel ? (
              <span className="ui-list__sub-label">
                {subLabel}
              </span>
            ) : null}
          </div>

          {dateValue ? (
            <DateDisplay
              className="ui-list__date"
              value={dateValue}
              variant="compact"
            />
          ) : null}
        </div>

        {preview || meta ? (
          <div className="ui-list__content">
            {preview ? (
              <p className="ui-list__preview">
                {preview}
              </p>
            ) : (
              <span className="ui-list__preview-spacer" />
            )}

            {meta ? (
              <div className="ui-list__meta">
                {meta}
              </div>
            ) : null}
          </div>
        ) : null}
      </div>
    </article>
  );
}

export default function List({
  children,
  className = "",
  ...props
}: ListProps) {
  const classes = ["ui-list", className]
    .filter(Boolean)
    .join(" ");

  return (
    <div className={classes} {...props}>
      {children}
    </div>
  );
}