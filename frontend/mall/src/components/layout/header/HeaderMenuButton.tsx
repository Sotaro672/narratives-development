// frontend/src/components/layout/header/HeaderMenuButton.tsx
type HeaderMenuButtonProps = {
  menuOpen: boolean;
  onClick: () => void;
};

export default function HeaderMenuButton({
  menuOpen,
  onClick,
}: HeaderMenuButtonProps) {
  return (
    <button
      type="button"
      className={`header__menu-button ${
        menuOpen ? "header__menu-button--open" : ""
      }`}
      aria-label={menuOpen ? "メニューを閉じる" : "メニューを開く"}
      aria-expanded={menuOpen}
      onClick={onClick}
    >
      <span className="header__menu-icon" aria-hidden="true">
        <span className="header__menu-line header__menu-line--1" />
        <span className="header__menu-line header__menu-line--2" />
        <span className="header__menu-line header__menu-line--3" />
      </span>
    </button>
  );
}