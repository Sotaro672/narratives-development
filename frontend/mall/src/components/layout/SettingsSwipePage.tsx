// frontend/mall/src/components/layout/SettingsSwipePage.tsx

import type { ReactNode } from "react";
import { useNavigate } from "react-router-dom";

import { useMobilePortrait } from "../hooks/useMobilePortrait";
import Layout from "./Layout";
import MobileSwipeRightDismissPage from "./MobileSwipeRightDismissPage";

type SettingsFooterAction = {
  buttonLabel: string;
  disabled?: boolean;
  onButtonClick?: () => void | Promise<void>;
  buttonType?: "button" | "submit";
  buttonForm?: string;
};

type SettingsSwipePageProps = {
  title: string;
  children: ReactNode;
  footerAction?: SettingsFooterAction;
};

export default function SettingsSwipePage({
  title,
  children,
  footerAction,
}: SettingsSwipePageProps) {
  const navigate = useNavigate();
  const isMobilePortrait = useMobilePortrait();

  const handleDismiss = () => {
    navigate(-1);
  };

  const showActionFooter =
    isMobilePortrait &&
    Boolean(footerAction);

  return (
    <MobileSwipeRightDismissPage
      title={title}
      enabled={isMobilePortrait}
      dismissGestureEnabled={isMobilePortrait}
      onDismiss={handleDismiss}
    >
      <Layout
        title=""
        titleClickable={false}
        mode="mypage"
        showHeader={false}
        showFooter={showActionFooter}
        footerProps={
          showActionFooter && footerAction
            ? {
                variant: "action",
                buttonLabel: footerAction.buttonLabel,
                disabled: footerAction.disabled,
                onButtonClick: footerAction.onButtonClick,
                buttonType: footerAction.buttonType,
                buttonForm: footerAction.buttonForm,
              }
            : undefined
        }
      >
        {children}
      </Layout>
    </MobileSwipeRightDismissPage>
  );
}