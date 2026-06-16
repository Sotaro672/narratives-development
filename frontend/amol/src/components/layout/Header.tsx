// frontend/amol/src/components/layout/Header.tsx
import "./header.css";
import "../../styles/settings-page.css";

import { useHeaderController } from "./header/useHeaderController";
import HeaderActions from "./header/HeaderActions";
import HeaderDesktopNavigation from "./header/HeaderDesktopNavigation";
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

        <HeaderDesktopNavigation />

        <HeaderActions actions={actions} />
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