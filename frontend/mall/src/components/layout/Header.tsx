// frontend/amol/src/components/layout/Header.tsx

import "./header.css";
import "../../styles/settings-page.css";

import { useHeaderController } from "./header/useHeaderController";
import HeaderActions from "./header/HeaderActions";
import HeaderDesktopNavigation from "./header/HeaderDesktopNavigation";
import HeaderSettingsPanel from "./header/HeaderSettingsPanel";
import type { HeaderProps } from "./header/types";

export default function Header(props: HeaderProps) {
  const {
    displayTitle,
    handleTitleClick,
    settingsOpen,
    shouldShowSettingsButton,
    closeSettings,
    actions,
  } = useHeaderController(props);

  const hasDirectActionButton =
    !!props.actionButtonLabel && typeof props.onActionButtonClick === "function";

  const mergedActions = {
    ...actions,
    hasActionButton: actions.hasActionButton || hasDirectActionButton,
    actionButtonLabel: props.actionButtonLabel ?? actions.actionButtonLabel,
    onActionButtonClick: props.onActionButtonClick ?? actions.onActionButtonClick,
    actionButtonDisabled:
      props.actionButtonDisabled ?? actions.actionButtonDisabled,
    shouldShowSettingsButton: hasDirectActionButton
      ? false
      : actions.shouldShowSettingsButton,
  };

  const shouldRenderSettingsPanel =
    shouldShowSettingsButton && !hasDirectActionButton;

  return (
    <header className="header">
      <div className="header__inner">
        <div className="header__left">
          {props.titleClickable === false ? (
            <span className="header__title header__title-text">
              {displayTitle}
            </span>
          ) : (
            <button
              type="button"
              className="header__title header__title-button"
              onClick={handleTitleClick}
            >
              {displayTitle}
            </button>
          )}
        </div>

        <HeaderDesktopNavigation
          showPublicNavigation={props.mode === "landing"}
        />

        <HeaderActions actions={mergedActions} />
      </div>

      {shouldRenderSettingsPanel ? (
        <HeaderSettingsPanel
          settingsOpen={settingsOpen}
          closeSettings={closeSettings}
        />
      ) : null}
    </header>
  );
}