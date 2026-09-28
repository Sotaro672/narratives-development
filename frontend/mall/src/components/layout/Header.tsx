// frontend/mall/src/components/layout/Header.tsx

import { ChevronLeft } from "lucide-react";

import "./header.css";
import "../../styles/settings-page.css";

import IconButton from "../ui/IconButton";
import HeaderActions from "./header/HeaderActions";
import HeaderDesktopNavigation from "./header/HeaderDesktopNavigation";
import HeaderSettingsPanel from "./header/HeaderSettingsPanel";
import type { HeaderProps } from "./header/types";
import { useHeaderController } from "./header/useHeaderController";

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

  const hasMobileContent = props.mobileContent != null;

  const mergedActions = {
    ...actions,
    hasActionButton: actions.hasActionButton || hasDirectActionButton,
    actionButtonLabel: props.actionButtonLabel ?? actions.actionButtonLabel,
    onActionButtonClick:
      props.onActionButtonClick ?? actions.onActionButtonClick,
    actionButtonDisabled:
      props.actionButtonDisabled ?? actions.actionButtonDisabled,
    shouldShowSettingsButton: hasDirectActionButton
      ? false
      : actions.shouldShowSettingsButton,
  };

  const shouldRenderSettingsPanel =
    shouldShowSettingsButton && !hasDirectActionButton;

  const headerClassName = [
    "header",
    hasMobileContent ? "header--with-mobile-content" : "",
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <header className={headerClassName}>
      <div className="header__inner">
        <div className="header__left">
          {hasBackButton ? (
            <IconButton
              variant="ghost"
              size="md"
              aria-label={props.backButtonLabel ?? "戻る"}
              title={props.backButtonLabel ?? "戻る"}
              onClick={() => {
                void props.onBackButtonClick?.();
              }}
            >
              <ChevronLeft
                size={24}
                strokeWidth={2}
                aria-hidden="true"
              />
            </IconButton>
          ) : null}

          <div className="header__title-container">
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

          {hasMobileContent ? (
            <div className="header__mobile-content">
              {props.mobileContent}
            </div>
          ) : null}
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