// frontend/amol/src/components/layout/Layout.tsx
import type { ReactNode } from "react";

import Header from "./Header";
import FooterNav from "./FooterNav";
import { WALLET_PATH } from "../../lib/navigation";
import "./layout.css";

type LayoutMode = "default" | "signin" | "mypage" | "landing";
type HeaderMode = "default" | "signin" | "landing";

type FooterProps =
  | {
      variant?: "default";
      renderMode?: "bottom" | "sidebar";
      onNavigate?: () => void;
    }
  | {
      variant: "action";
      buttonLabel: string;
      disabled?: boolean;
      onButtonClick: () => void | Promise<void>;
    }
  | {
      variant: "commentAction";
      value: string;
      placeholder?: string;
      buttonLabel: string;
      disabled?: boolean;
      posting?: boolean;
      onChange: (value: string) => void;
      onSubmit: () => void | Promise<void>;
    }
  | {
      variant: "reviewAction";
      value: string;
      rating: number;
      placeholder?: string;
      buttonLabel: string;
      disabled?: boolean;
      posting?: boolean;
      onChange: (value: string) => void;
      onRatingChange: (rating: number) => void;
      onSubmit: () => void | Promise<void>;
    };

type LayoutProps = {
  title: string;
  children: ReactNode;
  showBackButton?: boolean;
  mode?: LayoutMode;
  backTo?: string;
  showFooter?: boolean;
  showHeader?: boolean;
  showEditButton?: boolean;
  hideHamburgerMenu?: boolean;
  hideSettingsButton?: boolean;
  mainClassName?: string;
  disableFooterPaddingOnDesktop?: boolean;

  actionButtonLabel?: string;
  onActionButtonClick?: () => void | Promise<void>;
  actionButtonDisabled?: boolean;

  secondaryActionButtonLabel?: string;
  onSecondaryActionButtonClick?: () => void | Promise<void>;
  secondaryActionButtonDisabled?: boolean;

  showCartButton?: boolean;
  cartButtonLabel?: string;
  onCartButtonClick?: () => void | Promise<void>;
  cartButtonDisabled?: boolean;

  footerProps?: FooterProps;
};

export default function Layout({
  title,
  children,
  showBackButton = false,
  mode = "default",
  backTo = WALLET_PATH,
  showFooter,
  showHeader = true,
  showEditButton = false,
  hideHamburgerMenu = false,
  hideSettingsButton = false,
  mainClassName,
  disableFooterPaddingOnDesktop = false,

  actionButtonLabel,
  onActionButtonClick,
  actionButtonDisabled = false,

  secondaryActionButtonLabel,
  onSecondaryActionButtonClick,
  secondaryActionButtonDisabled = false,

  showCartButton = false,
  cartButtonLabel = "カート",
  onCartButtonClick,
  cartButtonDisabled = false,

  footerProps,
}: LayoutProps) {
  const shouldShowFooter = showFooter ?? mode === "mypage";
  const headerMode: HeaderMode = mode === "mypage" ? "default" : mode;
  const isActionFooter = shouldShowFooter && footerProps?.variant === "action";

  return (
    <div className="layout-shell">
      {showHeader ? (
        <Header
          title={title}
          showBackButton={showBackButton}
          mode={headerMode}
          backTo={backTo}
          showEditButton={showEditButton}
          hideHamburgerMenu={hideHamburgerMenu}
          hideSettingsButton={hideSettingsButton}
          actionButtonLabel={actionButtonLabel}
          onActionButtonClick={onActionButtonClick}
          actionButtonDisabled={actionButtonDisabled}
          secondaryActionButtonLabel={secondaryActionButtonLabel}
          onSecondaryActionButtonClick={onSecondaryActionButtonClick}
          secondaryActionButtonDisabled={secondaryActionButtonDisabled}
          showCartButton={showCartButton}
          cartButtonLabel={cartButtonLabel}
          onCartButtonClick={onCartButtonClick}
          cartButtonDisabled={cartButtonDisabled}
        />
      ) : null}

      <main
        className={[
          "layout-main",
          mainClassName ?? "",
          !showHeader ? "layout-main--without-header" : "",
          shouldShowFooter && !isActionFooter
            ? "layout-main--with-footer"
            : "",
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
        <FooterNav {...(footerProps ?? { variant: "default" })} />
      ) : null}
    </div>
  );
}