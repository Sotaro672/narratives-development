// frontend/src/components/layout/header/types.ts
export type HeaderProps = {
  title?: string;
  showBackButton?: boolean;
  backTo?: string;
  mode?: "default" | "landing" | "signin";
  showEditButton?: boolean;
  hideHamburgerMenu?: boolean;
  hideSettingsButton?: boolean;

  onBackButtonClick?: () => void | Promise<void>;

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
  cartItemCount?: number;
};

export type HeaderActionState = {
  hasActionButton: boolean;
  actionButtonLabel: string;
  onActionButtonClick?: () => void | Promise<void>;
  actionButtonDisabled: boolean;

  hasSecondaryActionButton: boolean;
  secondaryActionButtonLabel: string;
  onSecondaryActionButtonClick?: () => void | Promise<void>;
  secondaryActionButtonDisabled: boolean;

  shouldShowCartButton: boolean;
  cartButtonLabel: string;
  onCartButtonClick?: () => void | Promise<void>;
  cartButtonDisabled: boolean;
  cartItemCount: number;

  shouldShowLoginButton: boolean;
  shouldShowRoomCopyButton: boolean;
  shouldShowEditButton: boolean;
  shouldShowSettingsButton: boolean;
  copyButtonLabel: string;
  toggleSettings: () => void;
};