// frontend/console/brand/src/presentation/pages/brandDetail.tsx

import { Upload, X } from "lucide-react";

import "../styles/brand.css";

import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";

import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  CardLabel,
} from "../../../../shell/src/shared/ui/card";

import { Input } from "../../../../shell/src/shared/ui/input";

import { useBrandDetail } from "../hook/useBrandDetail";
import { ManagerCard } from "./components/ManagerCard";

export default function BrandDetail() {
  const {
    brand,
    handleBack,
    isEditing,
    draft,
    setDraft,
    handleEdit,
    handleCancelEdit,
    handleSave,
    loading,
    error,
    brandIconInputRef,
    brandBackgroundInputRef,
    brandIconFile,
    brandBackgroundFile,
    brandIconPreviewUrl,
    brandBackgroundPreviewUrl,
    handlePickBrandIcon,
    handlePickBrandBackground,
    handleBrandIconChange,
    handleBrandBackgroundChange,
    handleClearBrandIcon,
    handleClearBrandBackground,
  } = useBrandDetail();

  const hero = (
    <Card>
      <CardContent>
        {loading ? (
          <div className="text-sm text-muted-foreground text-left py-6">
            読み込み中...
          </div>
        ) : error ? (
          <div className="text-sm text-red-600 whitespace-pre-wrap text-left py-6">
            {error.message}
          </div>
        ) : (
          <div className="brand-hero">
            <div className="brand-hero__cover">
              {brandBackgroundPreviewUrl ? (
                <img
                  src={brandBackgroundPreviewUrl}
                  alt="Brand Background"
                  className="brand-hero__cover-image"
                  onClick={() => isEditing && handlePickBrandBackground()}
                  style={{ cursor: isEditing ? "pointer" : "default" }}
                />
              ) : (
                <div
                  className={`brand-hero__cover-empty${isEditing ? " is-clickable" : ""}`}
                  onClick={() => isEditing && handlePickBrandBackground()}
                >
                  {isEditing ? "背景画像を選択" : "背景画像未設定"}
                </div>
              )}

              {isEditing && (
                <input
                  ref={brandBackgroundInputRef}
                  type="file"
                  accept="image/*"
                  style={{ display: "none" }}
                  onChange={handleBrandBackgroundChange}
                />
              )}
            </div>

            {isEditing && (
              <div className="brand-hero__toolbar brand-hero__toolbar--cover">
                <button
                  type="button"
                  className="brand-hero__action-btn"
                  onClick={handlePickBrandBackground}
                >
                  <Upload size={16} />
                  背景画像をアップロード
                </button>
                {(brandBackgroundFile || draft.brandBackgroundImage) && (
                  <button
                    type="button"
                    className="brand-hero__action-btn"
                    onClick={handleClearBrandBackground}
                  >
                    <X size={16} />
                    取り消す
                  </button>
                )}
              </div>
            )}

            <div className="brand-hero__header">
              <div className="brand-hero__avatar-wrap">
                <div className="brand-hero__avatar">
                  {brandIconPreviewUrl ? (
                    <img
                      src={brandIconPreviewUrl}
                      alt="Brand Icon"
                      className="brand-hero__avatar-image"
                      onClick={() => isEditing && handlePickBrandIcon()}
                      style={{ cursor: isEditing ? "pointer" : "default" }}
                    />
                  ) : (
                    <div
                      className={`brand-hero__avatar-empty${isEditing ? " is-clickable" : ""}`}
                      onClick={() => isEditing && handlePickBrandIcon()}
                    >
                      {isEditing ? "アイコンを選択" : "アイコン未設定"}
                    </div>
                  )}

                  {isEditing && (
                    <input
                      ref={brandIconInputRef}
                      type="file"
                      accept="image/*"
                      style={{ display: "none" }}
                      onChange={handleBrandIconChange}
                    />
                  )}
                </div>

                {isEditing && (
                  <div className="brand-hero__toolbar brand-hero__toolbar--avatar">
                    <button
                      type="button"
                      className="brand-hero__action-btn brand-hero__action-btn--plain"
                      onClick={handlePickBrandIcon}
                    >
                      <Upload size={16} />
                      アイコンをアップロード
                    </button>
                    {(brandIconFile || draft.brandIcon) && (
                      <button
                        type="button"
                        className="brand-hero__action-btn brand-hero__action-btn--plain"
                        onClick={handleClearBrandIcon}
                      >
                        <X size={16} />
                        取り消す
                      </button>
                    )}
                  </div>
                )}
              </div>

              <div className="brand-hero__meta">
                <div className="brand-hero__title">{brand.name}</div>
                <div className="brand-hero__sub">
                  {brand.managerName || "責任者未設定"}
                </div>
                <div className="brand-hero__sub">
                  {brand.websiteUrl ? brand.websiteUrl : "Webサイト未設定"}
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
            <div className="text-sm text-muted-foreground text-left">
              読み込み中...
            </div>
          ) : error ? (
            <div className="text-sm text-red-600 whitespace-pre-wrap text-left">
              {error.message}
            </div>
          ) : (
            <>
              <CardLabel>ブランド名</CardLabel>
              {!isEditing ? (
                <div className="brand-view-plain">{brand.name}</div>
              ) : (
                <Input
                  value={draft.name}
                  placeholder="ブランド名"
                  onChange={(e) =>
                    setDraft((prev) => ({ ...prev, name: e.target.value }))
                  }
                  className="brand-detail__input"
                />
              )}

              <CardLabel>説明</CardLabel>
              {!isEditing ? (
                <div className="brand-detail__desc-box">{brand.description}</div>
              ) : (
                <textarea
                  value={draft.description}
                  placeholder="説明"
                  onChange={(e) =>
                    setDraft((prev) => ({
                      ...prev,
                      description: e.target.value,
                    }))
                  }
                  className="brand-detail__textarea"
                />
              )}

              <CardLabel>WebサイトURL</CardLabel>
              {!isEditing ? (
                <div className="brand-view-plain">
                  {brand.websiteUrl ? brand.websiteUrl : "（未設定）"}
                </div>
              ) : (
                <Input
                  value={draft.websiteUrl}
                  placeholder="https://example.com"
                  onChange={(e) =>
                    setDraft((prev) => ({
                      ...prev,
                      websiteUrl: e.target.value,
                    }))
                  }
                  className="brand-detail__input"
                />
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
        managerName={brand.managerName}
        managerId={brand.managerId}
        registeredAt={brand.registeredAt}
        updatedAt={brand.updatedAt}
        mode={isEditing ? "edit" : "view"}
      />
    </div>
  );

  return (
    <PageStyle
      layout="grid-2"
      title={`${brand.name}`}
      onBack={handleBack}
      onEdit={!isEditing ? handleEdit : undefined}
      onSave={isEditing ? handleSave : undefined}
      onCancel={isEditing ? handleCancelEdit : undefined}
      className={isEditing ? "brand-detail is-edit" : "brand-detail is-view"}
    >
      {[left, right]}
    </PageStyle>
  );
}