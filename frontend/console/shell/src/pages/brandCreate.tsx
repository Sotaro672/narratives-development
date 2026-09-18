// frontend/console/shell/src/pages/brandCreate.tsx

import { Upload, X } from "lucide-react";

import PageStyle from "../layout/PageStyle/PageStyle";
import {
  Card,
  CardContent,
  CardHeader,
  CardInput,
  CardLabel,
  CardTitle,
} from "../shared/ui/card";
import IconCropper from "../shared/ui/icon-cropper";
import EntityIcon from "../shared/ui/icon";
import { Media } from "../shared/ui/media";
import Textarea from "../shared/ui/textarea";

import { AdminCard } from "../features/admin/presentation/components/AdminCard";
import { AccountSelectCard } from "../features/brand/presentation/components/accountSelectCard";
import BrandCreateProgressModal from "../features/brand/presentation/components/brandCreateProgressModal";
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

    hasBrandIconSelection,
    hasBrandBackgroundSelection,

    brandIconInputRef,
    brandBackgroundInputRef,

    brandIconFile,
    brandIconPreviewUrl,
    brandBackgroundPreviewUrl,

    brandIconCropPosition,
    brandIconCropScale,
    handleBrandIconCropPositionChange,
    handleBrandIconCropScaleChange,
    handleBrandIconCropViewportSizeChange,

    brandIconError,
    brandBackgroundImageError,

    handlePickBrandIcon,
    handlePickBrandBackground,

    handleBrandIconChange,
    handleBrandBackgroundChange,

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
    accountCandidates.find(
      (candidate) => candidate.id === accountId,
    )?.label ?? null;

  const isCroppingBrandIcon = Boolean(
    brandIconFile && brandIconPreviewUrl,
  );

  const left = (
    <div className="brand-create__column">
      <Card>
        <CardContent>
          <div className="brand-hero">
            <Media
              src={brandBackgroundPreviewUrl}
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
              onActivate={
                saving
                  ? undefined
                  : handlePickBrandBackground
              }
              disabled={saving}
            />

            <input
              ref={brandBackgroundInputRef}
              type="file"
              accept={brandImageAccept}
              hidden
              onChange={handleBrandBackgroundChange}
              disabled={saving}
            />

            <div className="brand-hero__toolbar brand-hero__toolbar--cover">
              <button
                type="button"
                className="brand-hero__action-btn"
                onClick={handlePickBrandBackground}
                disabled={saving}
              >
                <Upload size={16} />
                背景画像をアップロード
              </button>

              {hasBrandBackgroundSelection && (
                <button
                  type="button"
                  className="brand-hero__action-btn"
                  onClick={handleClearBrandBackground}
                  disabled={saving}
                >
                  <X size={16} />
                  取り消す
                </button>
              )}
            </div>

            {brandBackgroundImageError && (
              <p className="brand-create__error brand-create__error--media">
                {brandBackgroundImageError}
              </p>
            )}

            <div className="brand-hero__header">
              <div className="brand-hero__avatar-wrap">
                {isCroppingBrandIcon ? (
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
                    onClick={
                      saving
                        ? undefined
                        : handlePickBrandIcon
                    }
                    disabled={saving}
                  />
                )}

                <input
                  ref={brandIconInputRef}
                  type="file"
                  accept={brandImageAccept}
                  hidden
                  onChange={handleBrandIconChange}
                  disabled={saving}
                />

                <div className="brand-hero__toolbar brand-hero__toolbar--avatar">
                  <button
                    type="button"
                    className="brand-hero__action-btn brand-hero__action-btn--plain"
                    onClick={handlePickBrandIcon}
                    disabled={saving}
                  >
                    <Upload size={16} />
                    アイコンをアップロード
                  </button>

                  {hasBrandIconSelection && (
                    <button
                      type="button"
                      className="brand-hero__action-btn brand-hero__action-btn--plain"
                      onClick={handleClearBrandIcon}
                      disabled={saving}
                    >
                      <X size={16} />
                      取り消す
                    </button>
                  )}
                </div>

                {brandIconError && (
                  <p className="brand-create__error brand-create__error--media">
                    {brandIconError}
                  </p>
                )}
              </div>

              <div className="brand-hero__meta">
                <div className="brand-hero__title">
                  {displayBrandName}
                </div>

                <div className="brand-hero__sub">
                  {managerDisplayName || "責任者未設定"}
                </div>

                <div className="brand-hero__sub">
                  {displayWebsiteUrl}
                </div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>
            ブランド情報
          </CardTitle>
        </CardHeader>

        <CardContent>
          <CardLabel htmlFor="name">
            ブランド名（必須）
          </CardLabel>

          <CardInput
            id="name"
            placeholder="ブランド名"
            value={name}
            onChange={(event) =>
              setName(event.target.value)
            }
            disabled={saving}
          />

          {nameError && (
            <p className="brand-create__error brand-create__error--field">
              {nameError}
            </p>
          )}

          <CardLabel htmlFor="description">
            説明
          </CardLabel>

          <Textarea
            id="description"
            value={description}
            placeholder="ブランドの説明を入力してください"
            className="brand-create__textarea"
            onChange={(event) =>
              setDescription(event.target.value)
            }
            disabled={saving}
          />

          <CardLabel htmlFor="websiteUrl">
            WebサイトURL
          </CardLabel>

          <CardInput
            id="websiteUrl"
            placeholder="https://example.com"
            value={websiteUrl}
            onChange={(event) =>
              setWebsiteUrl(event.target.value)
            }
            disabled={saving}
          />
        </CardContent>
      </Card>
    </div>
  );

  const right = (
    <div className="brand-create__column">
      <AdminCard
        mode="edit"
        assigneeId={managerId}
        assigneeName={managerDisplayName || "未設定"}
        assigneeCandidates={managerCandidates}
        loadingMembers={loadingManagers}
        onSelectAssignee={handleSelectManager}
      />

      {managerIdError && (
        <p className="brand-create__error">
          {managerIdError}
        </p>
      )}

      <AccountSelectCard
        accountLabel={accountLabel}
        accountCandidates={accountCandidates}
        loadingAccounts={loadingAccounts}
        accountError={
          accountLoadError || accountIdError
        }
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
        onClose={
          progress.isBlockingNavigation
            ? undefined
            : onCloseProgress
        }
      />
    </>
  );
}