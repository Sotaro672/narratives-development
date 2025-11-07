// frontend/production/src/pages/productionCreate.tsx
import * as React from "react";
import { useNavigate } from "react-router-dom";
import PageStyle from "../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../admin/src/pages/AdminCard";

/**
 * ProductionCreate
 * - 生産計画の新規作成ページ
 * - 左ペインに生産情報フォーム、右ペインに管理情報(AdminCard)を配置
 */
export default function ProductionCreate() {
  const navigate = useNavigate();

  // ──────────────────────────────────────────────
  // 入力フォーム状態（プリフィルは空）
  // ──────────────────────────────────────────────
  const [productionName, setProductionName] = React.useState("");
  const [brand, setBrand] = React.useState("");
  const [quantity, setQuantity] = React.useState<number>(0);
  const [dueDate, setDueDate] = React.useState("");
  const [description, setDescription] = React.useState("");

  // 管理情報
  const [assignee, setAssignee] = React.useState("未設定");
  const [creator] = React.useState("現在のユーザー");
  const [createdAt] = React.useState(new Date().toLocaleDateString());

  // ハンドラ
  const onCreate = () => {
    alert("生産計画を作成しました（ダミー）");
    navigate("/production");
  };

  const onBack = () => navigate(-1);

  return (
    <PageStyle
      layout="grid-2"
      title="生産計画の作成"
      onBack={onBack}
      onSave={onCreate} // 保存ボタン → 作成ボタンとして利用
    >
      {/* --- 左ペイン（フォーム） --- */}
      <div className="production-form">
        <h2 className="section-title">生産情報</h2>

        <div className="form-group">
          <label>生産名</label>
          <input
            type="text"
            value={productionName}
            onChange={(e) => setProductionName(e.target.value)}
            placeholder="例: シルクブラウス第1ロット"
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
          <label>数量</label>
          <input
            type="number"
            value={quantity}
            onChange={(e) => setQuantity(Number(e.target.value))}
            placeholder="例: 100"
          />
        </div>

        <div className="form-group">
          <label>納期</label>
          <input
            type="date"
            value={dueDate}
            onChange={(e) => setDueDate(e.target.value)}
          />
        </div>

        <div className="form-group">
          <label>備考</label>
          <textarea
            rows={4}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="必要なメモや特記事項を入力"
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
