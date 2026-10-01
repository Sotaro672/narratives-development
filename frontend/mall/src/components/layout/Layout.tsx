// frontend/mall/src/components/layout/Layout.tsx

import type { ReactNode } from "react";

import FooterNav from "./FooterNav";
import Header from "./Header";
import "./layout.css";

type LayoutMode =
  | "default"
  | "signin"
  | "mypage"
  | "landing";

type HeaderMode =
  | "default"
  | "signin"
  | "landing";

type FooterProps =
  | {
      variant?: "default";
      renderMode?: "bottom" | "sidebar";
      onNavigate?: () => void;
      centerActionLabel?: string;
      centerActionDisabled?: boolean;
      onCenterActionClick?: () => void | Promise<void>;
    }
  | {
      variant: "action";
      buttonLabel: string;
      disabled?: boolean;
      buttonType?: "button" | "submit";
      buttonForm?: string;
      onButtonClick?: () => void | Promise<void>;
    };

type LayoutProps = {
  title: string;
  titleClickable?: boolean;
  children: ReactNode;
  mode?: LayoutMode;
  showFooter?: boolean;
  showHeader?: boolean;
  hideSettingsButton?: boolean;
  hideAnnouncementButton?: boolean;
  headerMobileContent?: ReactNode;
  mainClassName?: string;
  disableFooterPaddingOnDesktop?: boolean;

  showBackButton?: boolean;
  backButtonLabel?: string;
  onBackButtonClick?: () => void | Promise<void>;

  actionButtonLabel?: string;
  onActionButtonClick?: () => void | Promise<void>;
  actionButtonDisabled?: boolean;

  secondaryActionButtonLabel?: string;
  onSecondaryActionButtonClick?: () => void | Promise<void>;
  secondaryActionButtonDisabled?: boolean;

  tertiaryActionButtonLabel?: string;
  onTertiaryActionButtonClick?: () => void | Promise<void>;
  tertiaryActionButtonDisabled?: boolean;

  showCartButton?: boolean;
  cartButtonLabel?: string;
  onCartButtonClick?: () => void | Promise<void>;
  cartButtonDisabled?: boolean;
  cartItemCount?: number;

  footerProps?: FooterProps;
};

export default function Layout({
  title,
  titleClickable = true,
  children,
  mode = "default",
  showFooter,
  showHeader = true,
  hideSettingsButton = false,
  hideAnnouncementButton = false,
  headerMobileContent,
  mainClassName,
  disableFooterPaddingOnDesktop = false,

  showBackButton = false,
  backButtonLabel = "戻る",
  onBackButtonClick,

  actionButtonLabel,
  onActionButtonClick,
  actionButtonDisabled = false,

  secondaryActionButtonLabel,
  onSecondaryActionButtonClick,
  secondaryActionButtonDisabled = false,

  tertiaryActionButtonLabel,
  onTertiaryActionButtonClick,
  tertiaryActionButtonDisabled = false,

  showCartButton = false,
  cartButtonLabel = "カート",
  onCartButtonClick,
  cartButtonDisabled = false,
  cartItemCount,

  footerProps,
}: LayoutProps) {
  const shouldShowFooter =
    showFooter ??
    mode === "mypage";

  const headerMode: HeaderMode =
    mode === "mypage"
      ? "default"
      : mode;

  const isActionFooter =
    shouldShowFooter &&
    footerProps?.variant === "action";

  return (
    <div className="layout-shell">
      {showHeader ? (
        <Header
          title={title}
          titleClickable={titleClickable}
          mode={headerMode}
          hideSettingsButton={hideSettingsButton}
          hideAnnouncementButton={hideAnnouncementButton}
          mobileContent={headerMobileContent}
          showBackButton={showBackButton}
          backButtonLabel={backButtonLabel}
          onBackButtonClick={onBackButtonClick}
          actionButtonLabel={actionButtonLabel}
          onActionButtonClick={onActionButtonClick}
          actionButtonDisabled={actionButtonDisabled}
          secondaryActionButtonLabel={secondaryActionButtonLabel}
          onSecondaryActionButtonClick={onSecondaryActionButtonClick}
          secondaryActionButtonDisabled={secondaryActionButtonDisabled}
          tertiaryActionButtonLabel={tertiaryActionButtonLabel}
          onTertiaryActionButtonClick={onTertiaryActionButtonClick}
          tertiaryActionButtonDisabled={tertiaryActionButtonDisabled}
          showCartButton={showCartButton}
          cartButtonLabel={cartButtonLabel}
          onCartButtonClick={onCartButtonClick}
          cartButtonDisabled={cartButtonDisabled}
          cartItemCount={cartItemCount}
        />
      ) : null}

      <main
        className={[
          "layout-main",
          mainClassName ?? "",
          !showHeader ? "layout-main--without-header" : "",
          shouldShowFooter && !isActionFooter ? "layout-main--with-footer" : "",
          isActionFooter ? "layout-main--with-action-footer" : "",
          disableFooterPaddingOnDesktop
            ? "layout-main--disable-footer-padding-desktop"
            : "",
        ]
          .filter(Boolean)
          .join(" ")}
      >
        {children}
      </main>

      {shouldShowFooter ? (
        <FooterNav
          {...(
            footerProps ?? {
              variant: "default",
            }
          )}
        />
      ) : null}
    </div>
  );
}