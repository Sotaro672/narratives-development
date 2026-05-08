//frontend\amol\src\pages\AvatarPage.tsx
import "../styles/page-layout.css";
import "../styles/form.css";
import "../styles/avatar-create-page.css";

import Layout from "../components/layout/Layout";
import Button from "../components/ui/Button";
import Input from "../components/ui/Input";
import Textbox from "../components/ui/Textbox";
import { useAvatarCreatePage } from "../features/avatar/hooks/useAvatarCreatePage";

export default function AvatarPage() {
  const vm = useAvatarCreatePage();

  const handleSaveClick = () => {
    void vm.save();
  };

  const handleSignOutClick = () => {
    void vm.signOut();
  };

  const isCreateMode = vm.mode === "create";
  const isEditMode = vm.mode === "edit";

  return (
    <Layout
      title={vm.pageTitle}
      showBackButton={isEditMode}
      backTo="/wallet"
      actionButtonLabel={vm.saveButtonLabel}
      onActionButtonClick={handleSaveClick}
      actionButtonDisabled={vm.saving || vm.loading}
      secondaryActionButtonLabel={isCreateMode ? "サインアウト" : undefined}
      onSecondaryActionButtonClick={isCreateMode ? handleSignOutClick : undefined}
      secondaryActionButtonDisabled={vm.saving || vm.loading}
      showFooter
      footerProps={{
        variant: "action",
        buttonLabel: vm.saveButtonLabel,
        disabled: vm.saving || vm.loading,
        onButtonClick: handleSaveClick,
      }}
    >
      <section className="page-section avatar-create-page-section">
        <div className="form-block avatar-create-form-block">
          <div className="avatar-create-icon-block">
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

            <div className="avatar-create-icon-preview">
              {vm.iconPreviewUrl ? (
                <img
                  src={vm.iconPreviewUrl}
                  alt="選択したアバターアイコン"
                  className="avatar-create-icon-image"
                  onError={vm.handleIconPreviewError}
                />
              ) : (
                <span className="avatar-create-icon-placeholder">
                  アイコン未選択
                </span>
              )}
            </div>

            <div className="avatar-create-icon-actions">
              <Button
                variant="secondary"
                onClick={vm.openIconPicker}
                disabled={vm.saving || vm.loading}
              >
                画像を選択
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

          {vm.msg ? (
            <div
              className={
                vm.isSuccessMessage
                  ? "avatar-create-message avatar-create-message--ok"
                  : "avatar-create-message avatar-create-message--info"
              }
            >
              {vm.msg}
            </div>
          ) : null}
        </div>
      </section>
    </Layout>
  );
}