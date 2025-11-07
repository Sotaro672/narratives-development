// frontend/tokenBlueprint/src/pages/tokenBlueprintCreate.tsx
import * as React from "react";
import { useNavigate } from "react-router-dom";
import PageStyle from "../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../admin/src/pages/AdminCard";

/**
 * TokenBlueprintCreate
 * - トークン設計の新規作成ページ
 * - 左ペイン: トークン情報入力フォーム
 * - 右ペイン: 管理情報(AdminCard)
 */
export default function TokenBlueprintCreate() {
  const navigate = useNavigate();

  // ──────────────────────────────────────────────
  // 入力フォーム状態（プリフィルは空）
  // ──────────────────────────────────────────────
  const [tokenName, setTokenName] = React.useState("");
  const [symbol, setSymbol] = React.useState("");
  const [brand, setBrand] = React.useState("");
  const [manager, setManager] = React.useState("");
  const [description, setDescription] = React.useState("");

  // 管理情報
  const [assignee, setAssignee] = React.useState("未設定");
  const [creator] = React.useState("現在のユーザー");
  const [createdAt] = React.useState(new Date().toLocaleDateString());

  // ハンドラ
  const onCreate = () => {
    alert("トークン設計を作成しました（ダミー）");
    navigate("/tokenBlueprint");
  };

  const onBack = () => navigate(-1);

  return (
    <PageStyle
      layout="grid-2"
      title="トークン設計の作成"
      onBack={onBack}
      onSave={onCreate} // 保存ボタン → 作成ボタンとして利用
    >
      {/* --- 左ペイン（トークン設計フォーム） --- */}
      <div className="token-blueprint-form">
        <h2 className="section-title">トークン情報</h2>

        <div className="form-group">
          <label>トークン名</label>
          <input
            type="text"
            value={tokenName}
            onChange={(e) => setTokenName(e.target.value)}
            placeholder="例: SILK Premium Token"
          />
        </div>

        <div className="form-group">
          <label>シンボル</label>
          <input
            type="text"
            value={symbol}
            onChange={(e) => setSymbol(e.target.value)}
            placeholder="例: SILK"
          />
        </div>

        <div className="form-group">
          <label>ブランド</label>
          <input
            type="text"
            value={brand}
            onChange={(e) => setBrand(e.target.value)}
            placeholder="例: LUMINA Fashion"
          />
        </div>

        <div className="form-group">
          <label>担当者</label>
          <input
            type="text"
            value={manager}
            onChange={(e) => setManager(e.target.value)}
            placeholder="例: 佐藤 美咲"
          />
        </div>

        <div className="form-group">
          <label>説明</label>
          <textarea
            rows={4}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="トークンの用途や概要を入力"
          />
        </div>
      </div>

      {/* --- 右ペイン（管理情報カード） --- */}
      <AdminCard
        title="管理情報"
        assigneeName={assignee}
        createdByName={creator}
        createdAt={createdAt}
        onEditAssignee={() => setAssignee("変更済み担当者")}
        onClickAssignee={() => console.log("Assignee clicked:", assignee)}
        onClickCreatedBy={() => console.log("CreatedBy clicked:", creator)}
      />
    </PageStyle>
  );
}
