// frontend/console/shell/src/pages/brandCreate.tsx

import PageStyle from "../layout/PageStyle/PageStyle";
import {
  Card,
  CardContent,
  CardHeader,
  CardInput,
  CardLabel,
  CardTitle,
} from "../shared/ui/card";
import { ErrorMessage } from "../shared/ui/error";
import IconCropper from "../shared/ui/icon-cropper";
import EntityIcon from "../shared/ui/icon";
import { Media } from "../shared/ui/media";
import MediaUploader from "../shared/ui/mediaUploader";
import Textarea from "../shared/ui/textarea";

import { AdminCard } from "../features/admin/presentation/components/AdminCard";
import { AccountSelectCard } from "../features/brand/presentation/components/accountSelectCard";
import BrandCreateProgressModal from "../features/brand/presentation/components/brandProgressModal";
import { useBrandCreate } from "../features/brand/presentation/hook/useBrandCreate";

import "../styles/brand.css";

export default function BrandCreate() {
  const {
    accountId,
    accountIdError,
    accountCandidates,
    loadingAccounts,
    accountLoadError,
    name,
    setName,
    nameError,
    description,
    setDescription,
    websiteUrl,
    setWebsiteUrl,
    managerId,
    managerIdError,
    managerDisplayName,
    managerCandidates,
    loadingManagers,
    handleSelectManager,
    displayBrandName,
    displayWebsiteUrl,
    brandImageAccept,
    brandIconFile,
    brandBackgroundFile,
    brandIconPreviewUrl,
    brandBackgroundPreviewUrl,
    brandIconCropPosition,
    brandIconCropScale,
    handleBrandIconCropPositionChange,
    handleBrandIconCropScaleChange,
    handleBrandIconCropViewportSizeChange,
    brandIconError,
    brandBackgroundImageError,
    handleBrandIconFilesSelected,
    handleBrandBackgroundFilesSelected,
    handleClearBrandIcon,
    handleClearBrandBackground,
    saving,
    progress,
    progressOpen,
    onCloseProgress,
    handleBack,
    handleSave,
  } = useBrandCreate();

  const accountLabel =
    accountCandidates.find((candidate) => candidate.id === accountId)?.label ?? null;

  const isCroppingBrandIcon = Boolean(brandIconFile && brandIconPreviewUrl);

  const brandBackgroundItems = brandBackgroundPreviewUrl
    ? [
        {
          id: "brand-background",
          src: brandBackgroundPreviewUrl,
          name: brandBackgroundFile?.name ?? "ブランド背景画像",
          alt: "ブランド背景画像",
          contentType: brandBackgroundFile?.type,
        },
      ]
    : [];

  const brandIconItems = brandIconPreviewUrl
    ? [
        {
          id: "brand-icon",
          src: brandIconPreviewUrl,
          name: brandIconFile?.name ?? "ブランドアイコン",
          alt: "ブランドアイコン",
          contentType: brandIconFile?.type,
        },
      ]
    : [];

  const left = (
    <div className="page-column">
      <Card>
        <CardContent>
          <div className="brand-hero">
            <MediaUploader
              items={brandBackgroundItems}
              accept={brandImageAccept}
              variant="single"
              pickerVariant="button"
              title={null}
              showCount={false}
              showFileNames={false}
              previewFramed={false}
              pickerLabel="背景画像をアップロード"
              replaceLabel="背景画像を変更"
              disabled={saving}
              className="brand-hero__background-uploader"
              renderPreview={(item, { openPicker }) => (
                <Media
                  src={item.src}
                  type="image"
                  alt="ブランド背景画像"
                  variant="cover"
                  fit="cover"
                  bordered={false}
                  onActivate={saving ? undefined : openPicker}
                  disabled={saving}
                />
              )}
              renderEmpty={({ openPicker }) => (
                <Media
                  type="image"
                  alt="ブランド背景画像"
                  variant="cover"
                  fit="cover"
                  bordered={false}
                  emptyText="背景画像を選択"
                  emptyDescription={
                    saving
                      ? undefined
                      : "クリックして背景画像を選択できます"
                  }
                  onActivate={saving ? undefined : openPicker}
                  disabled={saving}
                />
              )}
              onFilesSelected={handleBrandBackgroundFilesSelected}
              onRemove={() => handleClearBrandBackground()}
            />

            {brandBackgroundImageError && (
              <ErrorMessage
                as="p"
                size="xs"
                className="brand-create__error--media"
              >
                {brandBackgroundImageError}
              </ErrorMessage>
            )}

            <div className="brand-hero__header">
              <div className="brand-hero__avatar-wrap">
                <MediaUploader
                  items={brandIconItems}
                  accept={brandImageAccept}
                  variant="single"
                  pickerVariant="button"
                  title={null}
                  showCount={false}
                  showFileNames={false}
                  previewFramed={false}
                  pickerLabel="アイコンをアップロード"
                  replaceLabel="アイコンを変更"
                  disabled={saving}
                  className="brand-hero__avatar-uploader"
                  renderPreview={(_item, { openPicker }) =>
                    isCroppingBrandIcon ? (
                      <IconCropper
                        src={brandIconPreviewUrl}
                        position={brandIconCropPosition}
                        scale={brandIconCropScale}
                        onPositionChange={handleBrandIconCropPositionChange}
                        onScaleChange={handleBrandIconCropScaleChange}
                        onViewportSizeChange={handleBrandIconCropViewportSizeChange}
                        alt="ブランドアイコンの切り抜きプレビュー"
                        disabled={saving}
                      />
                    ) : (
                      <EntityIcon
                        src={brandIconPreviewUrl}
                        name={displayBrandName}
                        alt="ブランドアイコン"
                        size="fluid"
                        className="brand-hero__avatar"
                        imageClassName="brand-hero__avatar-image"
                        fallbackClassName="brand-hero__avatar-empty"
                        fallback="アイコンを選択"
                        onClick={saving ? undefined : openPicker}
                        disabled={saving}
                      />
                    )
                  }
                  renderEmpty={({ openPicker }) => (
                    <EntityIcon
                      name={displayBrandName}
                      alt="ブランドアイコン"
                      size="fluid"
                      className="brand-hero__avatar"
                      imageClassName="brand-hero__avatar-image"
                      fallbackClassName="brand-hero__avatar-empty"
                      fallback="アイコンを選択"
                      onClick={saving ? undefined : openPicker}
                      disabled={saving}
                    />
                  )}
                  onFilesSelected={handleBrandIconFilesSelected}
                  onRemove={() => handleClearBrandIcon()}
                />

                {brandIconError && (
                  <ErrorMessage
                    as="p"
                    size="xs"
                    className="brand-create__error--media"
                  >
                    {brandIconError}
                  </ErrorMessage>
                )}
              </div>

              <div className="brand-hero__meta">
                <div className="brand-hero__title">{displayBrandName}</div>
                <div className="brand-hero__sub">
                  {managerDisplayName || "責任者未設定"}
                </div>
                <div className="brand-hero__sub">{displayWebsiteUrl}</div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>ブランド情報</CardTitle>
        </CardHeader>

        <CardContent>
          <CardLabel htmlFor="name">ブランド名（必須）</CardLabel>

          <CardInput
            id="name"
            placeholder="ブランド名"
            value={name}
            onChange={(event) => setName(event.target.value)}
            disabled={saving}
          />

          {nameError && (
            <ErrorMessage
              as="p"
              size="xs"
              className="brand-create__error--field"
            >
              {nameError}
            </ErrorMessage>
          )}

          <CardLabel htmlFor="description">説明</CardLabel>

          <Textarea
            id="description"
            value={description}
            placeholder="ブランドの説明を入力してください"
            className="brand-create__textarea"
            onChange={(event) => setDescription(event.target.value)}
            disabled={saving}
          />

          <CardLabel htmlFor="websiteUrl">WebサイトURL</CardLabel>

          <CardInput
            id="websiteUrl"
            placeholder="https://example.com"
            value={websiteUrl}
            onChange={(event) => setWebsiteUrl(event.target.value)}
            disabled={saving}
          />
        </CardContent>
      </Card>
    </div>
  );

  const right = (
    <div className="page-column">
      <AdminCard
        mode="edit"
        assigneeId={managerId}
        assigneeName={managerDisplayName || "未設定"}
        assigneeCandidates={managerCandidates}
        loadingMembers={loadingManagers}
        onSelectAssignee={handleSelectManager}
      />

      {managerIdError && (
        <ErrorMessage as="p" size="xs">
          {managerIdError}
        </ErrorMessage>
      )}

      <AccountSelectCard
        accountLabel={accountLabel}
        accountCandidates={accountCandidates}
        loadingAccounts={loadingAccounts}
        accountError={accountLoadError || accountIdError}
      />
    </div>
  );

  return (
    <>
      <PageStyle
        layout="grid-2"
        title="ブランド登録"
        onBack={handleBack}
        onSave={handleSave}
        isSaving={saving}
      >
        {[left, right]}
      </PageStyle>

      <BrandCreateProgressModal
        open={progressOpen}
        progress={progress}
        onClose={progress.isBlockingNavigation ? undefined : onCloseProgress}
      />
    </>
  );
}