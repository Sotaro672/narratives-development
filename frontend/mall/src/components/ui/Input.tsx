// frontend/mall/src/components/ui/Input.tsx

import React, { forwardRef, useId } from "react";
import "./input.css";

export type InputSize = "md" | "lg";

type InputProps = Omit<React.InputHTMLAttributes<HTMLInputElement>, "size"> & {
  label?: string;
  error?: string;
  helperText?: string;
  fullWidth?: boolean;
  size?: InputSize;
};

const Input = forwardRef<HTMLInputElement, InputProps>(
  (
    {
      id,
      label,
      error,
      helperText,
      fullWidth = true,
      size = "md",
      className = "",
      disabled = false,
      required = false,
      ...props
    },
    ref,
  ) => {
    const generatedId = useId();
    const inputId = id ?? generatedId;
    const helperId = helperText ? `${inputId}-helper` : undefined;
    const errorId = error ? `${inputId}-error` : undefined;
    const describedBy = [helperId, errorId].filter(Boolean).join(" ") || undefined;

    return (
      <div className={`input-field ${fullWidth ? "input-field--full" : ""}`}>
        {label ? (
          <label className="input-field__label" htmlFor={inputId}>
            {label}
            {required ? <span className="input-field__required">*</span> : null}
          </label>
        ) : null}

        <input
          {...props}
          id={inputId}
          ref={ref}
          disabled={disabled}
          required={required}
          aria-invalid={Boolean(error)}
          aria-describedby={describedBy}
          className={[
            "input-field__control",
            size !== "md" ? `input-field__control--${size}` : "",
            error ? "input-field__control--error" : "",
            disabled ? "input-field__control--disabled" : "",
            className,
          ]
            .filter(Boolean)
            .join(" ")}
        />

        {helperText && !error ? (
          <p id={helperId} className="input-field__helper">
            {helperText}
          </p>
        ) : null}

        {error ? (
          <p id={errorId} className="input-field__error">
            {error}
          </p>
        ) : null}
      </div>
    );
  },
);

Input.displayName = "Input";

export default Input;
export type { InputProps };