//frontend\src\components\ui\Item.tsx
import "./item.css";

type ItemProps = {
  label: string;
  onClick?: () => void;
  danger?: boolean;
  rightText?: string;
};

export default function Item({
  label,
  onClick,
  danger = false,
  rightText = "›",
}: ItemProps) {
  return (
    <li className={`item${danger ? " item--danger" : ""}`}>
      <button type="button" className="item__button" onClick={onClick}>
        <span className="item__cell item__label">{label}</span>
        <span className="item__cell item__right" aria-hidden="true">
          {rightText}
        </span>
      </button>
    </li>
  );
}