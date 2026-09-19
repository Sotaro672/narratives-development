// frontend/console/shell/src/pages/brandDetail.tsx

import { Upload, X } from "lucide-react";

import "../styles/brand.css";

import PageStyle from "../layout/PageStyle/PageStyle";

import {
  Card,
  CardContent,
  CardHeader,
  CardLabel,
  CardTitle,
} from "../shared/ui/card";
import IconCropper from "../shared/ui/icon-cropper";
import EntityIcon from "../shared/ui/icon";
import { Input } from "../shared/ui/input";
import { Media } from "../shared/ui/media";
import Textarea from "../shared/ui/textarea";

import { useBrandDetail } from "../features/brand/presentation/hook/useBrandDetail";
import { ManagerCard } from "../features/brand/presentation/components/ManagerCard";
import { AccountSelectCard } from "../features/brand/presentation/components/accountSelectCard";
import BrandCreateProgressModal from "../features/brand/presentation/components/brandProgressModal";

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

    brandIconInputRef,
    brandBackgroundInputRef,

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

    handlePickBrandIcon,
    handlePickBrandBackground,

    handleBrandIconChange,
    handleBrandBackgroundChange,

    handleClearBrandIcon,
    handleClearBrandBackground,
  } = useBrandDetail();

  const canEditImage = isEditing && !saving;

  const isCroppingBrandIcon = Boolean(
    isEditing &&
    brandIconFile &&
    brandIconPreviewUrl,
  );

  const accountLabel =
    accountCandidates.find(
      (candidate) => candidate.id === brand.accountId,
    )?.label ?? null;

  const displayBrandName = isEditing
    ? draft.name || "ブランド名未入力"
    : brand.name || "ブランド名未設定";

  const hero = (
    <Card>
      <CardContent>
        {loading ? (
          <div className="brand-detail__state brand-detail__state--padded">
            読み込み中...
          </div>
        ) : error && !isEditing ? (
          <div className="brand-detail__state brand-detail__state--padded brand-detail__state--error brand-detail__state--pre-wrap">
            {error.message}
          </div>
        ) : (
          <div className="brand-hero">
            <Media
              src={brandBackgroundPreviewUrl}
              type="image"
              alt="ブランド背景画像"
              variant="cover"
              fit="cover"
              bordered={false}
              emptyText={
                isEditing
                  ? "背景画像を選択"
                  : "背景画像未設定"
              }
              emptyDescription={
                canEditImage
                  ? "クリックして背景画像を選択できます"
                  : undefined
              }
              onActivate={
                canEditImage
                  ? handlePickBrandBackground
                  : undefined
              }
              disabled={saving}
            />

            {isEditing && (
              <input
                ref={brandBackgroundInputRef}
                type="file"
                accept={brandImageAccept}
                hidden
                onChange={handleBrandBackgroundChange}
                disabled={saving}
              />
            )}

            {isEditing && (
              <>
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

                  {(brandBackgroundFile || draft.brandBackgroundImage) && (
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
                  <p className="brand-detail__media-error">
                    {brandBackgroundImageError}
                  </p>
                )}
              </>
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
                    fallback={
                      isEditing
                        ? "アイコンを選択"
                        : "アイコン未設定"
                    }
                    onClick={
                      canEditImage
                        ? handlePickBrandIcon
                        : undefined
                    }
                    disabled={saving}
                  />
                )}

                {isEditing && (
                  <input
                    ref={brandIconInputRef}
                    type="file"
                    accept={brandImageAccept}
                    hidden
                    onChange={handleBrandIconChange}
                    disabled={saving}
                  />
                )}

                {isEditing && (
                  <>
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

                      {(brandIconFile || draft.brandIcon) && (
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
                      <p className="brand-detail__media-error">
                        {brandIconError}
                      </p>
                    )}
                  </>
                )}
              </div>

              <div className="brand-hero__meta">
                <div className="brand-hero__title">
                  {displayBrandName}
                </div>

                <div className="brand-hero__sub">
                  {isEditing
                    ? editingManagerName
                    : brand.memberName || "責任者未設定"}
                </div>

                <div className="brand-hero__sub">
                  {isEditing
                    ? draft.websiteUrl || "Webサイト未設定"
                    : brand.websiteUrl || "Webサイト未設定"}
                </div>
              </div>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );

  const left = (
    <div className="brand-detail__column">
      {hero}

      <Card>
        <CardHeader>
          <CardTitle>
            基本情報
          </CardTitle>
        </CardHeader>

        <CardContent>
          {loading ? (
            <div className="brand-detail__state">
              読み込み中...
            </div>
          ) : (
            <>
              {error && isEditing && (
                <div className="brand-detail__form-error">
                  {error.message}
                </div>
              )}

              <CardLabel htmlFor="brand-name">
                ブランド名
              </CardLabel>

              {!isEditing ? (
                <div className="brand-view-plain">
                  {brand.name}
                </div>
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
                  className="brand-detail__input"
                  disabled={saving}
                />
              )}

              <CardLabel htmlFor="brand-description">
                説明
              </CardLabel>

              {!isEditing ? (
                <div className="brand-detail__desc-box">
                  {brand.description || "（未設定）"}
                </div>
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
                <div className="brand-view-plain">
                  {brand.websiteUrl || "（未設定）"}
                </div>
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
                  className="brand-detail__input"
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
    <div className="brand-detail__column">
      <ManagerCard
        managerName={
          isEditing
            ? editingManagerName
            : brand.memberName ?? ""
        }
        managerId={
          isEditing
            ? managerId
            : brand.managerId
        }
        managerCandidates={managerCandidates}
        loadingMembers={loadingMembers}
        onSelectManager={handleSelectManager}
        registeredAt={registeredAt}
        updatedAt={updatedAt}
        mode={
          isEditing
            ? "edit"
            : "view"
        }
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
        onEdit={
          !isEditing && !loading
            ? handleEdit
            : undefined
        }
        onSave={
          isEditing && !saving
            ? handleSave
            : undefined
        }
        onCancel={
          isEditing && !saving
            ? handleCancelEdit
            : undefined
        }
        className={
          isEditing
            ? "brand-detail is-edit"
            : "brand-detail is-view"
        }
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