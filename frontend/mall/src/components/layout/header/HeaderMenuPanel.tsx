// frontend/mall/src/components/layout/header/HeaderMenuPanel.tsx

import { useNavigate } from "react-router-dom";

import List, { ListItem } from "../../ui/List";
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
  const navigate = useNavigate();

  const handleNavigate = (to: string) => {
    closeMenu();
    navigate(to);
  };

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
          <nav aria-label="ページナビゲーション">
            <List>
              {publicHeaderNavigationItems.map((item) => (
                <ListItem
                  key={item.to}
                  label={item.label}
                  onClick={() => handleNavigate(item.to)}
                />
              ))}
            </List>
          </nav>
        )}
      </div>
    </>
  );
}