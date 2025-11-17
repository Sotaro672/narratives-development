import * as React from "react";
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
    brandName,
    setBrandName,
    brandCode,
    setBrandCode,
    category,
    setCategory,
    description,
    setDescription,
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
          <CardTitle>基本情報</CardTitle>
        </CardHeader>

        <CardContent>
          <CardLabel htmlFor="brandName">ブランド名</CardLabel>
          <CardInput
            id="brandName"
            placeholder="例：LUMINA Fashion"
            value={brandName}
            onChange={(e) => setBrandName(e.target.value)}
          />

          <CardLabel htmlFor="brandCode">ブランドコード</CardLabel>
          <CardInput
            id="brandCode"
            placeholder="例：LUMINA01"
            value={brandCode}
            onChange={(e) => setBrandCode(e.target.value)}
          />

          <CardLabel htmlFor="category">カテゴリ</CardLabel>
          <CardSelect
            id="category"
            value={category}
            onChange={(e) => setCategory(e.target.value)}
          >
            <option value="ファッション">ファッション</option>
            <option value="アクセサリー">アクセサリー</option>
            <option value="雑貨">雑貨</option>
            <option value="その他">その他</option>
          </CardSelect>

          <CardLabel htmlFor="description">説明</CardLabel>
          <textarea
            id="description"
            className="w-full h-28 border rounded-lg px-3 py-2 text-sm mt-1"
            placeholder="ブランドの説明を入力してください"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </CardContent>
      </Card>
    </PageStyle>
  );
}
