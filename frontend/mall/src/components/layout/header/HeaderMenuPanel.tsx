// frontend/mall/src/components/layout/header/HeaderMenuPanel.tsx

import { X } from "lucide-react";
import { useNavigate } from "react-router-dom";

import IconButton from "../../ui/IconButton";
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
        <div className="header__panel-close-row">
          <IconButton
            variant="ghost"
            size="md"
            className="header__panel-close-button"
            aria-label="メニューを閉じる"
            title="閉じる"
            onClick={closeMenu}
            tabIndex={menuOpen ? 0 : -1}
          >
            <X size={20} strokeWidth={1.8} aria-hidden="true" />
          </IconButton>
        </div>

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