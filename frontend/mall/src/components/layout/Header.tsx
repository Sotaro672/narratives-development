// frontend/mall/src/components/layout/Header.tsx

import { ChevronLeft } from "lucide-react";

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
    !!props.actionButtonLabel &&
    typeof props.onActionButtonClick === "function";

  const hasBackButton =
    props.showBackButton === true &&
    typeof props.onBackButtonClick === "function";

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
          {hasBackButton ? (
            <button
              type="button"
              className="header__back-button"
              aria-label={props.backButtonLabel ?? "戻る"}
              title={props.backButtonLabel ?? "戻る"}
              onClick={() => {
                void props.onBackButtonClick?.();
              }}
            >
              <ChevronLeft
                className="header__back-icon"
                size={24}
                strokeWidth={2}
                aria-hidden="true"
              />
            </button>
          ) : null}

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