//frontend\src\components\ui\Dropdown.tsx
import { useEffect, useRef, useState, type ReactNode } from "react";
import "./dropdown.css";

type DropdownItem<T extends string> = {
  value: T;
  label: string;
};

type DropdownProps<T extends string> = {
  buttonLabel: string;
  items: DropdownItem<T>[];
  selectedValue: T;
  onSelect: (value: T) => void;
  renderButton: (args: {
    isOpen: boolean;
    toggle: () => void;
  }) => ReactNode;
};

export default function Dropdown<T extends string>({
  items,
  selectedValue,
  onSelect,
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

  const toggle = () => {
    setIsOpen((prev) => !prev);
  };

  const handleSelect = (value: T) => {
    onSelect(value);
    setIsOpen(false);
  };

  return (
    <div className="ui-dropdown" ref={rootRef}>
      {renderButton({ isOpen, toggle })}

      {isOpen && (
        <div className="ui-dropdown__menu" role="menu">
          {items.map((item) => (
            <button
              key={item.value}
              type="button"
              className={`ui-dropdown__item ${
                selectedValue === item.value ? "is-selected" : ""
              }`}
              onClick={() => handleSelect(item.value)}
              role="menuitem"
            >
              {item.label}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}