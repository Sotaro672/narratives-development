// frontend/console/brand/src/presentation/pages/brandCreate.tsx
import { Upload, X } from "lucide-react";

import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";

import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  CardLabel,
  CardInput,
  CardSelect,
} from "../../../../shell/src/shared/ui/card";

import { useBrandCreate } from "../hook/useBrandCreate";

export default function BrandCreate() {
  const {
    name,
    setName,
    description,
    setDescription,
    websiteUrl,
    setWebsiteUrl,

    managerId,
    setManagerId,
    managerOptions,
    loadingManagers,
    managerError,

    displayBrandName,
    displayWebsiteUrl,
    managerDisplayName,
    hasBrandIconSelection,
    hasBrandBackgroundSelection,

    brandIconInputRef,
    brandBackgroundInputRef,
    brandIconPreviewUrl,
    brandBackgroundPreviewUrl,
    handlePickBrandIcon,
    handlePickBrandBackground,
    handleBrandIconChange,
    handleBrandBackgroundChange,
    handleClearBrandIcon,
    handleClearBrandBackground,

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
                    alt="Brand Background"
                    className="brand-hero__cover-image"
                    onClick={handlePickBrandBackground}
                    style={{ cursor: "pointer" }}
                  />
                ) : (
                  <div
                    className="brand-hero__cover-empty is-clickable"
                    onClick={handlePickBrandBackground}
                  >
                    背景画像を選択
                  </div>
                )}

                <input
                  ref={brandBackgroundInputRef}
                  type="file"
                  accept="image/*"
                  style={{ display: "none" }}
                  onChange={handleBrandBackgroundChange}
                />
              </div>

              <div className="brand-hero__toolbar brand-hero__toolbar--cover">
                <button
                  type="button"
                  className="brand-hero__action-btn"
                  onClick={handlePickBrandBackground}
                >
                  <Upload size={16} />
                  背景画像をアップロード
                </button>
                {hasBrandBackgroundSelection && (
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

              <div className="brand-hero__header">
                <div className="brand-hero__avatar-wrap">
                  <div className="brand-hero__avatar">
                    {brandIconPreviewUrl ? (
                      <img
                        src={brandIconPreviewUrl}
                        alt="Brand Icon"
                        className="brand-hero__avatar-image"
                        onClick={handlePickBrandIcon}
                        style={{ cursor: "pointer" }}
                      />
                    ) : (
                      <div
                        className="brand-hero__avatar-empty is-clickable"
                        onClick={handlePickBrandIcon}
                      >
                        アイコンを選択
                      </div>
                    )}

                    <input
                      ref={brandIconInputRef}
                      type="file"
                      accept="image/*"
                      style={{ display: "none" }}
                      onChange={handleBrandIconChange}
                    />
                  </div>

                  <div className="brand-hero__toolbar brand-hero__toolbar--avatar">
                    <button
                      type="button"
                      className="brand-hero__action-btn brand-hero__action-btn--plain"
                      onClick={handlePickBrandIcon}
                    >
                      <Upload size={16} />
                      アイコンをアップロード
                    </button>
                    {hasBrandIconSelection && (
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
                </div>

                <div className="brand-hero__meta">
                  <div className="brand-hero__title">{displayBrandName}</div>
                  <div className="brand-hero__sub">{managerDisplayName}</div>
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
              onChange={(e) => setName(e.target.value)}
            />

            <CardLabel htmlFor="description">説明</CardLabel>
            <textarea
              id="description"
              className="w-full h-28 border rounded-lg px-3 py-2 text-sm mt-1"
              placeholder="ブランドの説明を入力してください"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />

            <CardLabel htmlFor="websiteUrl">WebサイトURL</CardLabel>
            <CardInput
              id="websiteUrl"
              placeholder="https://example.com"
              value={websiteUrl}
              onChange={(e) => setWebsiteUrl(e.target.value)}
            />

            <CardLabel htmlFor="managerId" className="mt-6">
              ブランド責任者（必須）
            </CardLabel>

            <CardSelect
              id="managerId"
              value={managerId ?? ""}
              onChange={(e) => setManagerId(e.target.value || null)}
            >
              <option value="">未選択</option>

              {loadingManagers && <option value="">読み込み中...</option>}

              {!loadingManagers &&
                managerOptions.map((m) => (
                  <option key={m.id} value={m.id}>
                    {m.lastName || m.firstName
                      ? `${m.lastName ?? ""}${m.lastName && m.firstName ? " " : ""}${m.firstName ?? ""}` ||
                        m.email ||
                        m.id
                      : m.email || m.id}
                  </option>
                ))}
            </CardSelect>

            {managerError && (
              <p className="mt-1 text-xs text-red-500">{managerError}</p>
            )}
          </CardContent>
        </Card>
      </div>
    </PageStyle>
  );
}