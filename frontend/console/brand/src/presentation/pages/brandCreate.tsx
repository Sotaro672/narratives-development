// frontend/console/brand/src/ppresentation/pages/brandCreate.tsx
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
    companyId,
    name,
    setName,
    description,
    setDescription,
    websiteUrl,
    setWebsiteUrl,

    // managerId 選択用
    managerId,
    setManagerId,
    managerOptions,
    loadingManagers,
    managerError,
    formatLastFirst,

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
      <Card className="max-w-2xl">
        <CardHeader>
          <CardTitle>ブランド情報</CardTitle>
        </CardHeader>

        <CardContent>
          {/* ------------------------------- */}
          {/* ブランド名 */}
          {/* ------------------------------- */}
          <CardLabel htmlFor="name">ブランド名</CardLabel>
          <CardInput
            id="name"
            placeholder="例：LUMINA Fashion"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />

          {/* ------------------------------- */}
          {/* 説明 */}
          {/* ------------------------------- */}
          <CardLabel htmlFor="description">説明</CardLabel>
          <textarea
            id="description"
            className="w-full h-28 border rounded-lg px-3 py-2 text-sm mt-1"
            placeholder="ブランドの説明を入力してください"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />

          {/* ------------------------------- */}
          {/* WebサイトURL */}
          {/* ------------------------------- */}
          <CardLabel htmlFor="websiteUrl">WebサイトURL</CardLabel>
          <CardInput
            id="websiteUrl"
            placeholder="https://example.com"
            value={websiteUrl}
            onChange={(e) => setWebsiteUrl(e.target.value)}
          />

          {/* ======================================================== */}
          {/*    ▼▼▼ ここから下：メンバー選択欄（最下部へ移動） ▼▼▼    */}
          {/* ======================================================== */}

          <CardLabel htmlFor="managerId" className="mt-6">
            ブランド責任者（任意）
          </CardLabel>

          <CardSelect
            id="managerId"
            value={managerId ?? ""}
            onChange={(e) => setManagerId(e.target.value || null)}
          >
            <option value="">未選択</option>

            {loadingManagers && <option value="">読み込み中...</option>}

            {!loadingManagers &&
              managerOptions.map((m) => {
                const label =
                  formatLastFirst(m.lastName, m.firstName) ||
                  m.email ||
                  m.id;

                return (
                  <option key={m.id} value={m.id}>
                    {label}
                  </option>
                );
              })}
          </CardSelect>

          {managerError && (
            <p className="mt-1 text-xs text-red-500">{managerError}</p>
          )}

          {/* isActive, walletAddress は自動設定のため入力欄なし */}
        </CardContent>
      </Card>
    </PageStyle>
  );
}
