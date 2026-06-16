// frontend/src/components/layout/header/HeaderMenuPanel.tsx
import { Link } from "react-router-dom";

type HeaderMenuPanelProps = {
  menuOpen: boolean;
  closeMenu: () => void;
  shouldShowLandscapeSidebarMenuButton: boolean;
};

export default function HeaderMenuPanel({
  menuOpen,
  closeMenu,
}: HeaderMenuPanelProps) {
  return (
    <>
      <button
        type="button"
        className={`header__menu-backdrop ${
          menuOpen ? "header__menu-backdrop--open" : ""
        }`}
        onClick={closeMenu}
        aria-label="メニューを閉じる"
        aria-hidden={!menuOpen}
        tabIndex={menuOpen ? 0 : -1}
      />

      <div
        className={`header__menu-panel ${
          menuOpen ? "header__menu-panel--open" : ""
        }`}
        aria-hidden={!menuOpen}
      >
        <div className="header__menu-list">
          <Link
            to="/how-to-use"
            className="header__menu-link"
            onClick={closeMenu}
          >
            使い方
          </Link>

          <Link to="/faq" className="header__menu-link" onClick={closeMenu}>
            チーム
          </Link>

          <Link to="/terms" className="header__menu-link" onClick={closeMenu}>
            規約・ポリシー
          </Link>

          <Link to="/contact" className="header__menu-link" onClick={closeMenu}>
            お問い合わせ
          </Link>
        </div>
      </div>
    </>
  );
}