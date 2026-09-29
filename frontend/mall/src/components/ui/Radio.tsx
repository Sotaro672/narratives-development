// frontend/mall/src/components/ui/Radio.tsx

import type {
  InputHTMLAttributes,
  ReactNode,
} from "react";

import "./Radio.css";

export type RadioVariant = "default" | "card";

type RadioProps = Omit<
  InputHTMLAttributes<HTMLInputElement>,
  "type"
> & {
  label: ReactNode;
  description?: ReactNode;
  error?: ReactNode;
  variant?: RadioVariant;
};

export default function Radio({
  label,
  description,
  error,
  variant = "default",
  className = "",
  disabled = false,
  id,
  ...props
}: RadioProps) {
  const classes = [
    "ui-radio",
    variant !== "default" ? `ui-radio--${variant}` : "",
    disabled ? "ui-radio--disabled" : "",
    error ? "ui-radio--error" : "",
    className,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <div className={classes}>
      <label className="ui-radio__label" htmlFor={id}>
        <input
          {...props}
          id={id}
          type="radio"
          disabled={disabled}
          className="ui-radio__input"
        />

        <span className="ui-radio__control" aria-hidden="true">
          <span className="ui-radio__dot" />
        </span>

        <span className="ui-radio__content">
          <span className="ui-radio__text">
            {label}
          </span>

          {description ? (
            <span className="ui-radio__description">
              {description}
            </span>
          ) : null}
        </span>
      </label>

      {error ? (
        <div className="ui-radio__error">
          {error}
        </div>
      ) : null}
    </div>
  );
}