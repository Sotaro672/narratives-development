// frontend/amol/src/components/layout/header/HeaderMenuPanel.tsx
import { Link } from "react-router-dom";

import FooterNav from "../FooterNav";
import { publicHeaderNavigationItems } from "./headerNavigationItems";

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
            {publicHeaderNavigationItems.map((item) => (
              <Link
                key={item.to}
                to={item.to}
                className="header__menu-link"
                onClick={closeMenu}
              >
                {item.label}
              </Link>
            ))}
          </div>
        )}
      </div>
    </>
  );
}