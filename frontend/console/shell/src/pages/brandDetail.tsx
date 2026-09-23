// frontend/console/shell/src/pages/brandDetail.tsx

import * as React from "react";

import "../styles/brand.css";

import PageStyle from "../layout/PageStyle/PageStyle";
import {
  Card,
  CardContent,
  CardHeader,
  CardLabel,
  CardTitle,
} from "../shared/ui/card";
import { ErrorMessage } from "../shared/ui/error";
import IconCropper from "../shared/ui/icon-cropper";
import EntityIcon from "../shared/ui/icon";
import { Input } from "../shared/ui/input";
import Loading from "../shared/ui/loading";
import { Media } from "../shared/ui/media";
import MediaUploader from "../shared/ui/mediaUploader";
import Preview from "../shared/ui/preview";
import Stack from "../shared/ui/stack";
import { Text } from "../shared/ui/text";
import Textarea from "../shared/ui/textarea";

import { AdminCard } from "../features/admin/presentation/components/AdminCard";
import { AccountSelectCard } from "../features/brand/presentation/components/accountSelectCard";
import BrandCreateProgressModal from "../features/brand/presentation/components/brandProgressModal";
import { useBrandDetail } from "../features/brand/presentation/hook/useBrandDetail";

export default function BrandDetail() {
  const {
    brand,
    registeredAt,
    updatedAt,
    handleBack,
    isEditing,
    draft,
    setDraft,
    handleEdit,
    handleCancelEdit,
    handleSave,
    loading,
    saving,
    error,
    progress,
    progressOpen,
    onCloseProgress,
    managerId,
    managerCandidates,
    loadingMembers,
    editingManagerName,
    handleSelectManager,
    accountCandidates,
    loadingAccounts,
    accountError,
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
  } = useBrandDetail();

  const [backgroundPreviewOpen, setBackgroundPreviewOpen] =
    React.useState(false);
  const [iconPreviewOpen, setIconPreviewOpen] =
    React.useState(false);

  const isCroppingBrandIcon = Boolean(
    brandIconFile && brandIconPreviewUrl,
  );

  const accountLabel =
    accountCandidates.find((candidate) => candidate.id === brand.accountId)
      ?.label ?? null;

  const displayBrandName = isEditing
    ? draft.name || "ブランド名未入力"
    : brand.name || "ブランド名未設定";

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

  const hero = (
    <Card>
      <CardContent>
        {loading ? (
          <Loading
            variant="card"
            message="ブランド情報を読み込み中です..."
          />
        ) : error && !isEditing ? (
          <ErrorMessage className="brand-detail__state--padded">
            {error.message}
          </ErrorMessage>
        ) : isEditing ? (
          <Stack gap="md">
            <MediaUploader
              items={brandBackgroundItems}
              accept={brandImageAccept}
              variant="single"
              title={null}
              showCount={false}
              showPicker={false}
              showFileNames={false}
              previewFramed={false}
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
                className="brand-detail__media-error"
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
                  title={null}
                  showCount={false}
                  showPicker={false}
                  showFileNames={false}
                  previewFramed={false}
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
                    className="brand-detail__media-error"
                  >
                    {brandIconError}
                  </ErrorMessage>
                )}
              </div>

              <div className="brand-hero__meta">
                <div className="brand-hero__title">
                  {displayBrandName}
                </div>

                <div className="brand-hero__sub">
                  {editingManagerName || "責任者未設定"}
                </div>

                <div className="brand-hero__sub">
                  {draft.websiteUrl || "Webサイト未設定"}
                </div>
              </div>
            </div>
          </Stack>
        ) : (
          <div className="brand-hero">
            <Media
              src={brandBackgroundPreviewUrl}
              type="image"
              alt="ブランド背景画像"
              variant="cover"
              fit="cover"
              bordered={false}
              emptyText="背景画像未設定"
              onActivate={
                brandBackgroundPreviewUrl
                  ? () => setBackgroundPreviewOpen(true)
                  : undefined
              }
            />

            <div className="brand-hero__header">
              <div className="brand-hero__avatar-wrap">
                <EntityIcon
                  src={brandIconPreviewUrl}
                  name={displayBrandName}
                  alt="ブランドアイコン"
                  size="fluid"
                  className="brand-hero__avatar"
                  imageClassName="brand-hero__avatar-image"
                  fallbackClassName="brand-hero__avatar-empty"
                  fallback="アイコン未設定"
                  onClick={
                    brandIconPreviewUrl
                      ? () => setIconPreviewOpen(true)
                      : undefined
                  }
                />
              </div>

              <div className="brand-hero__meta">
                <div className="brand-hero__title">
                  {displayBrandName}
                </div>

                <div className="brand-hero__sub">
                  {brand.memberName || "責任者未設定"}
                </div>

                <div className="brand-hero__sub">
                  {brand.websiteUrl || "Webサイト未設定"}
                </div>
              </div>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );

  const left = (
    <div className="page-column">
      {hero}

      <Card>
        <CardHeader>
          <CardTitle>基本情報</CardTitle>
        </CardHeader>

        <CardContent>
          {loading ? (
            <Loading
              variant="card"
              message="基本情報を読み込み中です..."
            />
          ) : (
            <>
              {error && isEditing && (
                <ErrorMessage className="brand-detail__form-error">
                  {error.message}
                </ErrorMessage>
              )}

              <CardLabel htmlFor="brand-name">
                ブランド名
              </CardLabel>

              {!isEditing ? (
                <Text as="div" size="sm" wrap="anywhere">
                  {brand.name || "（未設定）"}
                </Text>
              ) : (
                <Input
                  id="brand-name"
                  value={draft.name}
                  placeholder="ブランド名"
                  onChange={(event) =>
                    setDraft((currentDraft) => ({
                      ...currentDraft,
                      name: event.target.value,
                    }))
                  }
                  disabled={saving}
                />
              )}

              <CardLabel htmlFor="brand-description">
                説明
              </CardLabel>

              {!isEditing ? (
                <Text as="div" size="sm" wrap="anywhere">
                  {brand.description || "（未設定）"}
                </Text>
              ) : (
                <Textarea
                  id="brand-description"
                  value={draft.description}
                  placeholder="説明"
                  onChange={(event) =>
                    setDraft((currentDraft) => ({
                      ...currentDraft,
                      description: event.target.value,
                    }))
                  }
                  disabled={saving}
                />
              )}

              <CardLabel htmlFor="brand-website-url">
                WebサイトURL
              </CardLabel>

              {!isEditing ? (
                <Text as="div" size="sm" wrap="anywhere">
                  {brand.websiteUrl || "（未設定）"}
                </Text>
              ) : (
                <Input
                  id="brand-website-url"
                  value={draft.websiteUrl}
                  placeholder="https://example.com"
                  onChange={(event) =>
                    setDraft((currentDraft) => ({
                      ...currentDraft,
                      websiteUrl: event.target.value,
                    }))
                  }
                  disabled={saving}
                />
              )}
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );

  const right = (
    <div className="page-column">
      <AdminCard
        assigneeLabel="責任者"
        assigneeName={
          isEditing
            ? editingManagerName
            : brand.memberName ?? ""
        }
        assigneeId={
          isEditing
            ? managerId ?? undefined
            : brand.managerId ?? undefined
        }
        assigneeCandidates={managerCandidates}
        loadingMembers={loadingMembers}
        onSelectAssignee={handleSelectManager}
        createdAt={registeredAt}
        updatedAt={updatedAt}
        mode={isEditing ? "edit" : "view"}
      />

      <AccountSelectCard
        accountLabel={accountLabel}
        accountCandidates={accountCandidates}
        loadingAccounts={loadingAccounts}
        accountError={accountError}
      />
    </div>
  );

  return (
    <>
      <PageStyle
        layout="grid-2"
        title={brand.name || "ブランド詳細"}
        onBack={handleBack}
        onEdit={!isEditing && !loading ? handleEdit : undefined}
        onSave={isEditing && !saving ? handleSave : undefined}
        onCancel={isEditing && !saving ? handleCancelEdit : undefined}
      >
        {[left, right]}
      </PageStyle>

      <BrandCreateProgressModal
        open={progressOpen}
        progress={progress}
        onClose={
          progress.isBlockingNavigation
            ? undefined
            : onCloseProgress
        }
      />

      <Preview
        open={backgroundPreviewOpen}
        src={brandBackgroundPreviewUrl}
        alt={`${displayBrandName}の背景画像`}
        onClose={() => setBackgroundPreviewOpen(false)}
      />

      <Preview
        open={iconPreviewOpen}
        src={brandIconPreviewUrl}
        alt={`${displayBrandName}のブランドアイコン`}
        onClose={() => setIconPreviewOpen(false)}
      />
    </>
  );
}