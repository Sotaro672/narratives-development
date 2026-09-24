//frontend\mall\src\components\ui\Checkbox.tsx
import type {
  InputHTMLAttributes,
  ReactNode,
} from "react";

import "./Checkbox.css";

type CheckboxProps = Omit<
  InputHTMLAttributes<HTMLInputElement>,
  "type"
> & {
  label: ReactNode;
  description?: ReactNode;
  error?: ReactNode;
};

export default function Checkbox({
  label,
  description,
  error,
  className = "",
  disabled = false,
  id,
  ...props
}: CheckboxProps) {
  const classes = [
    "ui-checkbox",
    disabled ? "ui-checkbox--disabled" : "",
    error ? "ui-checkbox--error" : "",
    className,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <div className={classes}>
      <label className="ui-checkbox__label" htmlFor={id}>
        <input
          {...props}
          id={id}
          type="checkbox"
          disabled={disabled}
          className="ui-checkbox__input"
        />

        <span className="ui-checkbox__control" aria-hidden="true">
          <span className="ui-checkbox__check" />
        </span>

        <span className="ui-checkbox__content">
          <span className="ui-checkbox__text">
            {label}
          </span>

          {description ? (
            <span className="ui-checkbox__description">
              {description}
            </span>
          ) : null}
        </span>
      </label>

      {error ? (
        <div className="ui-checkbox__error">
          {error}
        </div>
      ) : null}
    </div>
  );
}