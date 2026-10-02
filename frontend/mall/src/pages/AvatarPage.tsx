// frontend/mall/src/pages/AvatarPage.tsx

import { useNavigate } from "react-router-dom";

import "../styles/page-layout.css";
import "../styles/form.css";
import "../styles/avatar-create-page.css";

import { useMobilePortrait } from "../components/hooks/useMobilePortrait";
import Layout from "../components/layout/Layout";
import MobileSwipeDismissPage from "../components/layout/MobileSwipeDismissPage";
import Alert from "../components/ui/Alert";
import Button from "../components/ui/Button";
import Input from "../components/ui/Input";
import Textbox from "../components/ui/Textbox";
import AvatarCreateProgressModal from "../features/avatar/components/AvatarCreateProgressModal";
import AvatarIconCropper from "../features/avatar/components/AvatarIconCropper";
import { useAvatarCreatePage } from "../features/avatar/hooks/useAvatarCreatePage";

export default function AvatarPage() {
  const navigate = useNavigate();
  const isMobilePortrait = useMobilePortrait();
  const vm = useAvatarCreatePage();

  const handleSaveClick = () => {
    void vm.save();
  };

  const handleSignOutClick = () => {
    void vm.signOut();
  };

  const isCreateMode = vm.mode === "create";
  const isEditMode = vm.mode === "edit";
  const isMobileEditMode = isEditMode && isMobilePortrait;
  const isIconEditing = Boolean(vm.iconFile && vm.iconPreviewUrl);

  const dismissGestureEnabled =
    isMobileEditMode &&
    !vm.loading &&
    !vm.saving &&
    !vm.progress.isBlockingNavigation;

  const handleDismiss = () => {
    if (
      vm.loading ||
      vm.saving ||
      vm.progress.isBlockingNavigation
    ) {
      return;
    }

    navigate("/wallet", {
      replace: true,
    });
  };

  const content = (
    <Layout
      title="AMOL"
      showHeader={!isMobileEditMode}
      showFooter
      mainClassName={
        isMobileEditMode
          ? "avatar-create-page-main--mobile-edit"
          : undefined
      }
      secondaryActionButtonLabel={
        isCreateMode ? "サインアウト" : undefined
      }
      onSecondaryActionButtonClick={
        isCreateMode ? handleSignOutClick : undefined
      }
      secondaryActionButtonDisabled={vm.saving || vm.loading}
      footerProps={
        isMobileEditMode
          ? {
              variant: "action",
              buttonLabel: vm.saveButtonLabel,
              disabled:
                vm.saving ||
                vm.loading ||
                vm.progress.isBlockingNavigation,
              onButtonClick: handleSaveClick,
            }
          : undefined
      }
    >
      <section className="page-section avatar-create-page-section">
        <div className="form-block avatar-create-form-block">
          <div
            className="avatar-create-icon-block"
            data-mobile-swipe-dismiss-ignore={
              isIconEditing ? "true" : undefined
            }
          >
            <input
              ref={vm.fileInputRef}
              type="file"
              accept="image/*"
              className="avatar-create-file-input"
              onChange={(event) => {
                vm.pickIcon(event.target.files?.[0] ?? null);
                event.target.value = "";
              }}
            />

            {isIconEditing && vm.iconPreviewUrl ? (
              <AvatarIconCropper
                src={vm.iconPreviewUrl}
                position={vm.iconPosition}
                scale={vm.iconScale}
                onPositionChange={vm.setIconPosition}
                onScaleChange={vm.setIconScale}
                onViewportSizeChange={vm.setIconViewportSize}
                alt="選択したアバターアイコン"
                disabled={vm.saving || vm.loading}
              />
            ) : (
              <div className="avatar-create-icon-preview">
                {vm.iconPreviewUrl ? (
                  <img
                    src={vm.iconPreviewUrl}
                    alt="アバターアイコン"
                    className="avatar-create-icon-image"
                    onError={vm.handleIconPreviewError}
                  />
                ) : (
                  <span className="avatar-create-icon-placeholder">
                    アイコン未選択
                  </span>
                )}
              </div>
            )}

            <div className="avatar-create-icon-actions">
              <Button
                variant="secondary"
                onClick={vm.openIconPicker}
                disabled={vm.saving || vm.loading}
              >
                {vm.iconPreviewUrl ? "画像を変更" : "画像を選択"}
              </Button>

              {vm.iconPreviewUrl || vm.iconFile ? (
                <Button
                  variant="secondary"
                  onClick={vm.clearIcon}
                  disabled={vm.saving || vm.loading}
                >
                  画像を削除
                </Button>
              ) : null}
            </div>

            {vm.iconFileName ? (
              <p className="avatar-create-icon-meta">
                {vm.iconFileName}
                {vm.iconMimeType ? ` / ${vm.iconMimeType}` : ""}
              </p>
            ) : null}
          </div>

          <Input
            label="アバター名"
            type="text"
            placeholder="アバター名を入力"
            value={vm.avatarName}
            onChange={(event) => {
              vm.setAvatarName(event.target.value);
              vm.clearMessage();
            }}
            disabled={vm.saving || vm.loading}
            fullWidth
          />

          <Textbox
            label="プロフィール"
            placeholder="自己紹介を入力"
            value={vm.profile}
            onChange={(event) => {
              vm.setProfile(event.target.value);
              vm.clearMessage();
            }}
            disabled={vm.saving || vm.loading}
            fullWidth
          />

          <Input
            label="外部リンク"
            type="url"
            placeholder="https://example.com"
            value={vm.externalLink}
            onChange={(event) => {
              vm.setExternalLink(event.target.value);
              vm.clearMessage();
            }}
            disabled={vm.saving || vm.loading}
            fullWidth
          />

          {!isMobileEditMode ? (
            <div className="avatar-create-page-actions">
              <Button
                type="button"
                variant="primary"
                onClick={handleSaveClick}
                disabled={vm.saving || vm.loading}
                fullWidth
              >
                {vm.saveButtonLabel}
              </Button>
            </div>
          ) : null}

          {vm.msg ? (
            <Alert
              variant={vm.isSuccessMessage ? "success" : "info"}
              className="avatar-create-message"
            >
              {vm.msg}
            </Alert>
          ) : null}
        </div>
      </section>
    </Layout>
  );

  const page = isMobileEditMode ? (
    <MobileSwipeDismissPage
      dismissButtonAriaLabel="アバター編集を閉じる"
      dismissGestureEnabled={dismissGestureEnabled}
      onDismiss={handleDismiss}
    >
      {content}
    </MobileSwipeDismissPage>
  ) : (
    content
  );

  return (
    <>
      {page}

      <AvatarCreateProgressModal
        open={vm.progressOpen}
        progress={vm.progress}
        onClose={
          vm.progress.isBlockingNavigation
            ? undefined
            : vm.handleCloseProgress
        }
      />
    </>
  );
}