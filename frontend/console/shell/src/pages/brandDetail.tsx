// frontend/console/shell/src/pages/brandDetail.tsx

import { Upload, X } from "lucide-react";
import "../styles/brand.css";
import PageStyle from "../layout/PageStyle/PageStyle";

import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  CardLabel,
} from "../shared/ui/card";
import { Input } from "../shared/ui/input";

import { useBrandDetail } from "../features/brand/presentation/hook/useBrandDetail";
import { ManagerCard } from "../features/brand/presentation/components/ManagerCard";

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

    managerCandidates,
    loadingMembers,
    memberError,

    editingManagerName,
    handleSelectManager,

    brandImageAccept,

    brandIconInputRef,
    brandBackgroundInputRef,

    brandIconFile,
    brandBackgroundFile,

    brandIconPreviewUrl,
    brandBackgroundPreviewUrl,

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

  const hero = (
    <Card>
      <CardContent>
        {loading ? (
          <div className="py-6 text-left text-sm text-muted-foreground">
            読み込み中...
          </div>
        ) : error && !isEditing ? (
          <div className="py-6 text-left text-sm text-red-600 whitespace-pre-wrap">
            {error.message}
          </div>
        ) : (
          <div className="brand-hero">
            <div className="brand-hero__cover">
              {brandBackgroundPreviewUrl ? (
                <img
                  src={brandBackgroundPreviewUrl}
                  alt="ブランド背景画像"
                  className="brand-hero__cover-image"
                  onClick={canEditImage ? handlePickBrandBackground : undefined}
                  style={{ cursor: canEditImage ? "pointer" : "default" }}
                />
              ) : (
                <div
                  className={`brand-hero__cover-empty${canEditImage ? " is-clickable" : ""}`}
                  onClick={canEditImage ? handlePickBrandBackground : undefined}
                  style={{ cursor: canEditImage ? "pointer" : "default" }}
                >
                  {isEditing ? "背景画像を選択" : "背景画像未設定"}
                </div>
              )}

              {isEditing && (
                <input
                  ref={brandBackgroundInputRef}
                  type="file"
                  accept={brandImageAccept}
                  style={{ display: "none" }}
                  onChange={handleBrandBackgroundChange}
                  disabled={saving}
                />
              )}
            </div>

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
                  <p className="mt-2 text-xs text-red-500">
                    {brandBackgroundImageError}
                  </p>
                )}
              </>
            )}

            <div className="brand-hero__header">
              <div className="brand-hero__avatar-wrap">
                <div className="brand-hero__avatar">
                  {brandIconPreviewUrl ? (
                    <img
                      src={brandIconPreviewUrl}
                      alt="ブランドアイコン"
                      className="brand-hero__avatar-image"
                      onClick={canEditImage ? handlePickBrandIcon : undefined}
                      style={{ cursor: canEditImage ? "pointer" : "default" }}
                    />
                  ) : (
                    <div
                      className={`brand-hero__avatar-empty${canEditImage ? " is-clickable" : ""}`}
                      onClick={canEditImage ? handlePickBrandIcon : undefined}
                      style={{ cursor: canEditImage ? "pointer" : "default" }}
                    >
                      {isEditing ? "アイコンを選択" : "アイコン未設定"}
                    </div>
                  )}

                  {isEditing && (
                    <input
                      ref={brandIconInputRef}
                      type="file"
                      accept={brandImageAccept}
                      style={{ display: "none" }}
                      onChange={handleBrandIconChange}
                      disabled={saving}
                    />
                  )}
                </div>

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
                      <p className="mt-2 text-xs text-red-500">
                        {brandIconError}
                      </p>
                    )}
                  </>
                )}
              </div>

              <div className="brand-hero__meta">
                <div className="brand-hero__title">
                  {isEditing
                    ? draft.name || "ブランド名未入力"
                    : brand.name || "ブランド名未設定"}
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
    <div className="space-y-4">
      {hero}

      <Card>
        <CardHeader>
          <CardTitle>基本情報</CardTitle>
        </CardHeader>

        <CardContent>
          {loading ? (
            <div className="text-left text-sm text-muted-foreground">
              読み込み中...
            </div>
          ) : (
            <>
              {error && isEditing && (
                <div className="mb-4 text-left text-sm text-red-600 whitespace-pre-wrap">
                  {error.message}
                </div>
              )}

              <CardLabel htmlFor="brand-name">ブランド名</CardLabel>

              {!isEditing ? (
                <div className="brand-view-plain">{brand.name}</div>
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

              <CardLabel htmlFor="brand-description">説明</CardLabel>

              {!isEditing ? (
                <div className="brand-detail__desc-box">
                  {brand.description || "（未設定）"}
                </div>
              ) : (
                <textarea
                  id="brand-description"
                  value={draft.description}
                  placeholder="説明"
                  onChange={(event) =>
                    setDraft((currentDraft) => ({
                      ...currentDraft,
                      description: event.target.value,
                    }))
                  }
                  className="brand-detail__textarea"
                  disabled={saving}
                />
              )}

              <CardLabel htmlFor="brand-website-url">WebサイトURL</CardLabel>

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

              {saving && (
                <p className="mt-4 text-sm text-slate-500">
                  ブランド情報と画像を保存しています...
                </p>
              )}
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );

  const right = (
    <div className="space-y-4">
      <ManagerCard
        managerName={isEditing ? editingManagerName : brand.memberName ?? ""}
        managerId={isEditing ? draft.managerId : brand.managerId}
        managerCandidates={managerCandidates}
        loadingMembers={loadingMembers}
        memberError={memberError}
        onSelectManager={handleSelectManager}
        registeredAt={registeredAt}
        updatedAt={updatedAt}
        mode={isEditing ? "edit" : "view"}
      />
    </div>
  );

  return (
    <PageStyle
      layout="grid-2"
      title={brand.name || "ブランド詳細"}
      onBack={handleBack}
      onEdit={!isEditing && !loading ? handleEdit : undefined}
      onSave={isEditing && !saving ? handleSave : undefined}
      onCancel={isEditing && !saving ? handleCancelEdit : undefined}
      className={isEditing ? "brand-detail is-edit" : "brand-detail is-view"}
    >
      {[left, right]}
    </PageStyle>
  );
}