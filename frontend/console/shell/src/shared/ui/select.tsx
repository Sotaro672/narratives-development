// frontend/console/shell/src/shared/ui/select.tsx

import * as React from "react";
import {
  Check,
  ChevronDown,
} from "lucide-react";

import "./select.css";

function cn(...classes: Array<string | undefined | false | null>): string {
  return classes.filter(Boolean).join(" ");
}

export type SelectOption = {
  value: string;
  label: React.ReactNode;
  disabled?: boolean;
};

type SelectBaseProps = {
  options: SelectOption[];
  placeholder?: React.ReactNode;
  label?: React.ReactNode;
  emptyText?: React.ReactNode;
  className?: string;
  contentClassName?: string;
  size?: "sm" | "default";
  disabled?: boolean;
  renderValue?: (selectedOptions: SelectOption[]) => React.ReactNode;
  ariaLabel?: string;
};

type SingleSelectProps = SelectBaseProps & {
  multiple?: false;
  value: string;
  onChange: (value: string) => void;
};

type MultipleSelectProps = SelectBaseProps & {
  multiple: true;
  value: ReadonlySet<string>;
  onChange: (value: string, selected: boolean) => void;
};

export type SelectProps =
  | SingleSelectProps
  | MultipleSelectProps;

export function Select(props: SelectProps) {
  const {
    options,
    placeholder = "選択してください",
    label,
    emptyText = "選択可能な項目がありません。",
    className,
    contentClassName,
    size = "default",
    disabled = false,
    renderValue,
    ariaLabel,
  } = props;

  const [open, setOpen] = React.useState(false);
  const rootRef = React.useRef<HTMLDivElement | null>(null);
  const triggerId = React.useId();
  const listboxId = React.useId();

  const selectedOptions = React.useMemo(
    () =>
      options.filter((option) =>
        props.multiple
          ? props.value.has(option.value)
          : props.value === option.value,
      ),
    [options, props.multiple, props.value],
  );

  const hasSelection = selectedOptions.length > 0;

  React.useEffect(() => {
    if (disabled) {
      setOpen(false);
    }
  }, [disabled]);

  React.useEffect(() => {
    if (!open) {
      return;
    }

    const handleMouseDown = (event: MouseEvent) => {
      const target = event.target as Node;

      if (
        rootRef.current &&
        !rootRef.current.contains(target)
      ) {
        setOpen(false);
      }
    };

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault();
        setOpen(false);
      }
    };

    document.addEventListener("mousedown", handleMouseDown);
    document.addEventListener("keydown", handleKeyDown);

    return () => {
      document.removeEventListener("mousedown", handleMouseDown);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [open]);

  const handleOptionSelect = (
    option: SelectOption,
  ) => {
    if (disabled || option.disabled) {
      return;
    }

    if (props.multiple) {
      const selected = !props.value.has(option.value);
      props.onChange(option.value, selected);
      return;
    }

    props.onChange(option.value);
    setOpen(false);
  };

  const renderTriggerValue = () => {
    if (!hasSelection) {
      return placeholder;
    }

    if (renderValue) {
      return renderValue(selectedOptions);
    }

    if (props.multiple) {
      return `${selectedOptions.length}件選択`;
    }

    return selectedOptions[0]?.label ?? placeholder;
  };

  return (
    <div
      ref={rootRef}
      className={cn(
        "ui-select",
        disabled && "ui-select--disabled",
        className,
      )}
    >
      {label ? (
        <label
          htmlFor={triggerId}
          className="ui-select__label"
        >
          {label}
        </label>
      ) : null}

      <button
        id={triggerId}
        type="button"
        className={cn(
          "ui-select__trigger",
          `ui-select__trigger--${size}`,
          open && "ui-select__trigger--open",
        )}
        disabled={disabled}
        aria-label={ariaLabel}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-controls={listboxId}
        onClick={() => {
          if (!disabled) {
            setOpen((current) => !current);
          }
        }}
        onKeyDown={(event) => {
          if (
            !disabled &&
            (event.key === "ArrowDown" || event.key === "Enter")
          ) {
            event.preventDefault();
            setOpen(true);
          }
        }}
      >
        <span
          className={cn(
            "ui-select__value",
            !hasSelection && "ui-select__value--placeholder",
          )}
        >
          {renderTriggerValue()}
        </span>

        <ChevronDown
          className={cn(
            "ui-select__chevron",
            open && "ui-select__chevron--open",
          )}
          aria-hidden="true"
        />
      </button>

      {open && !disabled ? (
        <div
          id={listboxId}
          className={cn(
            "ui-select__content",
            contentClassName,
          )}
          role="listbox"
          aria-multiselectable={
            props.multiple ? true : undefined
          }
        >
          {options.length === 0 ? (
            <div className="ui-select__empty">
              {emptyText}
            </div>
          ) : (
            <div className="ui-select__list">
              {options.map((option) => {
                const isSelected = props.multiple
                  ? props.value.has(option.value)
                  : props.value === option.value;

                return (
                  <button
                    key={option.value}
                    type="button"
                    role="option"
                    aria-selected={isSelected}
                    disabled={option.disabled}
                    className={cn(
                      "ui-select__option",
                      isSelected && "ui-select__option--selected",
                    )}
                    onClick={() => {
                      handleOptionSelect(option);
                    }}
                  >
                    <span className="ui-select__option-label">
                      {option.label}
                    </span>

                    <span
                      className={cn(
                        "ui-select__check",
                        !isSelected && "ui-select__check--hidden",
                      )}
                      aria-hidden="true"
                    >
                      <Check />
                    </span>
                  </button>
                );
              })}
            </div>
          )}
        </div>
      ) : null}
    </div>
  );
}

export default Select;