// frontend/mall/src/components/ui/Dropdown.tsx

import { useEffect, useRef, useState, type ReactNode } from "react";

import "./dropdown.css";

type DropdownItem<T extends string> = {
  value: T;
  label: string;
};

type DropdownRenderButtonArgs = {
  isOpen: boolean;
  toggle: () => void;
  disabled: boolean;
};

type DropdownProps<T extends string> = {
  buttonLabel: string;
  items: DropdownItem<T>[];
  selectedValue: T;
  onSelect: (value: T) => void;
  disabled?: boolean;
  className?: string;
  buttonClassName?: string;
  renderButton?: (args: DropdownRenderButtonArgs) => ReactNode;
};

export default function Dropdown<T extends string>({
  buttonLabel,
  items,
  selectedValue,
  onSelect,
  disabled = false,
  className = "",
  buttonClassName = "",
  renderButton,
}: DropdownProps<T>) {
  const [isOpen, setIsOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (rootRef.current && !rootRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };

    document.addEventListener("mousedown", handleClickOutside);

    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, []);

  useEffect(() => {
    if (disabled) {
      setIsOpen(false);
    }
  }, [disabled]);

  const toggle = () => {
    if (disabled) {
      return;
    }

    setIsOpen((prev) => !prev);
  };

  const handleSelect = (value: T) => {
    if (disabled) {
      return;
    }

    onSelect(value);
    setIsOpen(false);
  };

  const classes = [
    "ui-dropdown",
    disabled ? "ui-dropdown--disabled" : "",
    className,
  ]
    .filter(Boolean)
    .join(" ");

  const triggerClasses = [
    "ui-dropdown__trigger",
    buttonClassName,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <div className={classes} ref={rootRef}>
      {renderButton ? (
        renderButton({
          isOpen,
          toggle,
          disabled,
        })
      ) : (
        <button
          type="button"
          className={triggerClasses}
          disabled={disabled}
          aria-expanded={isOpen}
          aria-haspopup="menu"
          onClick={toggle}
        >
          <span className="ui-dropdown__trigger-label">
            {buttonLabel}
          </span>

          <span
            className={[
              "ui-dropdown__trigger-icon",
              isOpen ? "ui-dropdown__trigger-icon--open" : "",
            ]
              .filter(Boolean)
              .join(" ")}
            aria-hidden="true"
          >
            ▼
          </span>
        </button>
      )}

      {isOpen ? (
        <div className="ui-dropdown__menu" role="menu">
          {items.map((item) => (
            <button
              key={item.value}
              type="button"
              className={[
                "ui-dropdown__item",
                selectedValue === item.value ? "is-selected" : "",
              ]
                .filter(Boolean)
                .join(" ")}
              onClick={() => handleSelect(item.value)}
              role="menuitem"
            >
              {item.label}
            </button>
          ))}
        </div>
      ) : null}
    </div>
  );
}