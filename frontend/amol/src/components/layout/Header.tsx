// frontend/src/components/layout/Header.tsx
import { Link } from "react-router-dom";

import "./header.css";
import "../../styles/settings-page.css";

import { useHeaderController } from "./header/useHeaderController";
import HeaderActions from "./header/HeaderActions";
import HeaderMenuButton from "./header/HeaderMenuButton";
import HeaderMenuPanel from "./header/HeaderMenuPanel";
import HeaderSettingsPanel from "./header/HeaderSettingsPanel";
import type { HeaderProps } from "./header/types";

export default function Header(props: HeaderProps) {
  const {
    displayTitle,
    handleTitleClick,
    menuOpen,
    settingsOpen,
    shouldShowMenuButton,
    shouldShowBackButton,
    shouldShowLandscapeSidebarMenuButton,
    shouldShowSettingsButton,
    closeMenu,
    closeSettings,
    handleBack,
    toggleMenu,
    actions,
  } = useHeaderController(props);

  return (
    <header className="header">
      <div className="header__inner">
        <div className="header__left">
          {shouldShowMenuButton ? (
            <HeaderMenuButton menuOpen={menuOpen} onClick={toggleMenu} />
          ) : null}

          {shouldShowBackButton ? (
            <button
              type="button"
              className="header__back-button"
              aria-label="戻る"
              onClick={handleBack}
            >
              ←
            </button>
          ) : null}

          <button
            type="button"
            className="header__title header__title-button"
            onClick={handleTitleClick}
          >
            {displayTitle}
          </button>
        </div>

        <div className="header__right">
          <nav
            className="header__desktop-nav"
            aria-label="ページナビゲーション"
          >
            <Link to="/how-to-use" className="header__desktop-nav-link">
              使い方
            </Link>

            <Link to="/faq" className="header__desktop-nav-link">
              チーム
            </Link>

            <Link to="/terms" className="header__desktop-nav-link">
              規約・ポリシー
            </Link>

            <Link to="/contact" className="header__desktop-nav-link">
              お問い合わせ
            </Link>
          </nav>

          <HeaderActions actions={actions} />
        </div>
      </div>

      {shouldShowMenuButton ? (
        <HeaderMenuPanel
          menuOpen={menuOpen}
          closeMenu={closeMenu}
          shouldShowLandscapeSidebarMenuButton={
            shouldShowLandscapeSidebarMenuButton
          }
        />
      ) : null}

      {shouldShowSettingsButton ? (
        <HeaderSettingsPanel
          settingsOpen={settingsOpen}
          closeSettings={closeSettings}
        />
      ) : null}
    </header>
  );
}