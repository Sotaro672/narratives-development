// frontend/console/shell/src/pages/brandCreate.tsx

import { Upload, X } from "lucide-react";

import PageStyle from "../layout/PageStyle/PageStyle";

import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  CardLabel,
  CardInput,
} from "../shared/ui/card";

import { AdminCard } from "../features/admin/presentation/components/AdminCard";
import { useBrandCreate } from "../features/brand/presentation/hook/useBrandCreate";

export default function BrandCreate() {
  const {
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

    saving,

    handleBack,
    handleSave,
  } = useBrandCreate();

  const left = (
    <div className="space-y-4">
      <Card>
        <CardContent>
          <div className="brand-hero">
            <div className="brand-hero__cover">
              {brandBackgroundPreviewUrl ? (
                <img
                  src={brandBackgroundPreviewUrl}
                  alt="ブランド背景画像"
                  className="brand-hero__cover-image"
                  onClick={
                    saving
                      ? undefined
                      : handlePickBrandBackground
                  }
                  style={{
                    cursor: saving
                      ? "default"
                      : "pointer",
                  }}
                />
              ) : (
                <button
                  type="button"
                  className="brand-hero__cover-empty is-clickable"
                  onClick={handlePickBrandBackground}
                  disabled={saving}
                >
                  背景画像を選択
                </button>
              )}

              <input
                ref={brandBackgroundInputRef}
                type="file"
                accept={brandImageAccept}
                style={{ display: "none" }}
                onChange={
                  handleBrandBackgroundChange
                }
                disabled={saving}
              />
            </div>

            <div className="brand-hero__toolbar brand-hero__toolbar--cover">
              <button
                type="button"
                className="brand-hero__action-btn"
                onClick={
                  handlePickBrandBackground
                }
                disabled={saving}
              >
                <Upload size={16} />
                背景画像をアップロード
              </button>

              {hasBrandBackgroundSelection && (
                <button
                  type="button"
                  className="brand-hero__action-btn"
                  onClick={
                    handleClearBrandBackground
                  }
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

            <div className="brand-hero__header">
              <div className="brand-hero__avatar-wrap">
                <div className="brand-hero__avatar">
                  {brandIconPreviewUrl ? (
                    <img
                      src={brandIconPreviewUrl}
                      alt="ブランドアイコン"
                      className="brand-hero__avatar-image"
                      onClick={
                        saving
                          ? undefined
                          : handlePickBrandIcon
                      }
                      style={{
                        cursor: saving
                          ? "default"
                          : "pointer",
                      }}
                    />
                  ) : (
                    <button
                      type="button"
                      className="brand-hero__avatar-empty is-clickable"
                      onClick={
                        handlePickBrandIcon
                      }
                      disabled={saving}
                    >
                      アイコンを選択
                    </button>
                  )}

                  <input
                    ref={brandIconInputRef}
                    type="file"
                    accept={brandImageAccept}
                    style={{
                      display: "none",
                    }}
                    onChange={
                      handleBrandIconChange
                    }
                    disabled={saving}
                  />
                </div>

                <div className="brand-hero__toolbar brand-hero__toolbar--avatar">
                  <button
                    type="button"
                    className="brand-hero__action-btn brand-hero__action-btn--plain"
                    onClick={
                      handlePickBrandIcon
                    }
                    disabled={saving}
                  >
                    <Upload size={16} />
                    アイコンをアップロード
                  </button>

                  {hasBrandIconSelection && (
                    <button
                      type="button"
                      className="brand-hero__action-btn brand-hero__action-btn--plain"
                      onClick={
                        handleClearBrandIcon
                      }
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
              </div>

              <div className="brand-hero__meta">
                <div className="brand-hero__title">
                  {displayBrandName}
                </div>

                <div className="brand-hero__sub">
                  {managerDisplayName ||
                    "責任者未設定"}
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
              setName(
                event.target.value,
              )
            }
            disabled={saving}
          />

          {nameError && (
            <p className="mt-1 text-xs text-red-500">
              {nameError}
            </p>
          )}

          <CardLabel htmlFor="description">
            説明
          </CardLabel>

          <textarea
            id="description"
            className="w-full h-28 border rounded-lg px-3 py-2 text-sm mt-1"
            placeholder="ブランドの説明を入力してください"
            value={description}
            onChange={(event) =>
              setDescription(
                event.target.value,
              )
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
              setWebsiteUrl(
                event.target.value,
              )
            }
            disabled={saving}
          />

          {saving && (
            <p className="mt-4 text-sm text-slate-500">
              ブランドを登録しています...
            </p>
          )}
        </CardContent>
      </Card>
    </div>
  );

  const right = (
    <div className="space-y-4">
      <AdminCard
        mode="edit"
        assigneeId={managerId}
        assigneeName={
          managerDisplayName ||
          "未設定"
        }
        assigneeCandidates={
          managerCandidates
        }
        loadingMembers={
          loadingManagers
        }
        onSelectAssignee={
          handleSelectManager
        }
      />

      {managerIdError && (
        <p className="text-xs text-red-500">
          {managerIdError}
        </p>
      )}
    </div>
  );

  return (
    <PageStyle
      layout="grid-2"
      title="ブランド登録"
      onBack={handleBack}
      onSave={handleSave}
      isSaving={saving}
    >
      {[left, right]}
    </PageStyle>
  );
}