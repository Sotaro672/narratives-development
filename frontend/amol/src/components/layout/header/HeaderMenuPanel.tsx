// frontend/src/components/layout/header/HeaderMenuPanel.tsx
import { Link } from "react-router-dom";

import FooterNav from "../FooterNav";

type HeaderMenuPanelProps = {
  menuOpen: boolean;
  closeMenu: () => void;
  shouldShowLandscapeSidebarMenuButton: boolean;
};

export default function HeaderMenuPanel({
  menuOpen,
  closeMenu,
  shouldShowLandscapeSidebarMenuButton,
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
        } ${
          shouldShowLandscapeSidebarMenuButton
            ? "header__menu-panel--sidebar"
            : ""
        }`}
        aria-hidden={!menuOpen}
      >
        {shouldShowLandscapeSidebarMenuButton ? (
          <FooterNav renderMode="sidebar" onNavigate={closeMenu} />
        ) : (
          <div className="header__menu-list">
            <Link to="/" className="header__menu-link" onClick={closeMenu}>
              トップ
            </Link>

            <Link
              to="/how-to-use"
              className="header__menu-link"
              onClick={closeMenu}
            >
              使い方
            </Link>

            <Link
              to="/faq"
              className="header__menu-link"
              onClick={closeMenu}
            >
              FAQ
            </Link>

            <Link
              to="/specified-commercial-transactions"
              className="header__menu-link"
              onClick={closeMenu}
            >
              特定商取引法に基づく表記
            </Link>

            <Link
              to="/terms"
              className="header__menu-link"
              onClick={closeMenu}
            >
              利用規約
            </Link>

            <Link
              to="/privacy-policy"
              className="header__menu-link"
              onClick={closeMenu}
            >
              プライバシーポリシー
            </Link>

            <Link
              to="/contact"
              className="header__menu-link"
              onClick={closeMenu}
            >
              お問い合わせ
            </Link>
          </div>
        )}
      </div>
    </>
  );
}