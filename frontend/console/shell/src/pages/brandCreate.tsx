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
  CardSelect,
} from "../shared/ui/card";

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
    setManagerId,
    managerIdError,
    managerOptions,
    loadingManagers,
    managerError,

    displayBrandName,
    displayWebsiteUrl,
    managerDisplayName,

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

  return (
    <PageStyle
      layout="single"
      title="ブランド登録"
      onBack={handleBack}
      onSave={handleSave}
    >
      <div className="space-y-4 max-w-2xl">
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
                  onChange={handleBrandBackgroundChange}
                  disabled={saving}
                />
              </div>

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
                        onClick={handlePickBrandIcon}
                        disabled={saving}
                      >
                        アイコンを選択
                      </button>
                    )}

                    <input
                      ref={brandIconInputRef}
                      type="file"
                      accept={brandImageAccept}
                      style={{ display: "none" }}
                      onChange={handleBrandIconChange}
                      disabled={saving}
                    />
                  </div>

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
                    {managerDisplayName}
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
            <CardTitle>ブランド情報</CardTitle>
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

            <CardLabel
              htmlFor="managerId"
              className="mt-6"
            >
              ブランド責任者（必須）
            </CardLabel>

            <CardSelect
              id="managerId"
              value={managerId ?? ""}
              onChange={(event) =>
                setManagerId(
                  event.target.value || null,
                )
              }
              disabled={loadingManagers || saving}
            >
              <option value="">未選択</option>

              {loadingManagers && (
                <option value="">
                  読み込み中...
                </option>
              )}

              {!loadingManagers &&
                managerOptions.map((manager) => {
                  const fullName =
                    manager.lastName ||
                    manager.firstName
                      ? `${manager.lastName ?? ""}${
                          manager.lastName &&
                          manager.firstName
                            ? " "
                            : ""
                        }${manager.firstName ?? ""}`
                      : "";

                  const label =
                    fullName ||
                    manager.email ||
                    manager.id;

                  return (
                    <option
                      key={manager.id}
                      value={manager.id}
                    >
                      {label}
                    </option>
                  );
                })}
            </CardSelect>

            {managerIdError && (
              <p className="mt-1 text-xs text-red-500">
                {managerIdError}
              </p>
            )}

            {managerError && (
              <p className="mt-1 text-xs text-red-500">
                {managerError}
              </p>
            )}

            {saving && (
              <p className="mt-4 text-sm text-slate-500">
                ブランドを登録しています...
              </p>
            )}
          </CardContent>
        </Card>
      </div>
    </PageStyle>
  );
}