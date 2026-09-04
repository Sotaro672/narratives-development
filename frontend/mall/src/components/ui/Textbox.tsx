// frontend/src/components/ui/Textbox.tsx
import React, { forwardRef, useId } from "react";
import "./textbox.css";

type TextboxProps = Omit<
  React.TextareaHTMLAttributes<HTMLTextAreaElement>,
  "size"
> & {
  label?: string;
  error?: string;
  helperText?: string;
  counterText?: string;
  fullWidth?: boolean;
};

const Textbox = forwardRef<HTMLTextAreaElement, TextboxProps>(
  (
    {
      id,
      label,
      error,
      helperText,
      counterText,
      fullWidth = true,
      className = "",
      disabled = false,
      required = false,
      ...props
    },
    ref
  ) => {
    const generatedId = useId();
    const textboxId = id ?? generatedId;
    const helperId = helperText ? `${textboxId}-helper` : undefined;
    const errorId = error ? `${textboxId}-error` : undefined;
    const counterId = counterText ? `${textboxId}-counter` : undefined;
    const describedBy =
      [helperId, errorId, counterId].filter(Boolean).join(" ") || undefined;

    return (
      <div className={`textbox-field ${fullWidth ? "textbox-field--full" : ""}`}>
        {label && (
          <label className="textbox-field__label" htmlFor={textboxId}>
            {label}
            {required && <span className="textbox-field__required">*</span>}
          </label>
        )}

        <textarea
          {...props}
          id={textboxId}
          ref={ref}
          disabled={disabled}
          required={required}
          aria-invalid={Boolean(error)}
          aria-describedby={describedBy}
          className={[
            "textbox-field__control",
            error ? "textbox-field__control--error" : "",
            disabled ? "textbox-field__control--disabled" : "",
            className,
          ]
            .filter(Boolean)
            .join(" ")}
        />

        {helperText && !error && (
          <p id={helperId} className="textbox-field__helper">
            {helperText}
          </p>
        )}

        {error && (
          <p id={errorId} className="textbox-field__error">
            {error}
          </p>
        )}

        {counterText && (
          <p id={counterId} className="textbox-field__counter">
            {counterText}
          </p>
        )}
      </div>
    );
  }
);

Textbox.displayName = "Textbox";

export default Textbox;
export type { TextboxProps };