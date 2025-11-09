//frontend\brand\src\pages\brandCreate.tsx
import * as React from "react";
import { useNavigate } from "react-router-dom"; // ← 追加
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

export default function BrandCreate() {
  const navigate = useNavigate(); // ← 追加

  const [brandName, setBrandName] = React.useState("");
  const [brandCode, setBrandCode] = React.useState("");
  const [category, setCategory] = React.useState("ファッション");
  const [description, setDescription] = React.useState("");

  // ─────────────────────────────────────────────
  // 戻るボタン（←）
  // ─────────────────────────────────────────────
  const handleBack = React.useCallback(() => {
    navigate(-1); // 一つ前のページへ戻る
  }, [navigate]);

  // ─────────────────────────────────────────────
  // 保存ボタン処理（モック）
  // ─────────────────────────────────────────────
  const handleSave = () => {
    console.log("保存:", {
      brandName,
      brandCode,
      category,
      description,
    });
    alert("ブランド情報を保存しました（モック）");
  };

  // ─────────────────────────────────────────────
  // JSX
  // ─────────────────────────────────────────────
  return (
    <PageStyle
      layout="single"
      title="ブランド登録"
      onBack={handleBack}  // ← 追加
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
