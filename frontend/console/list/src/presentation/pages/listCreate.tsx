// frontend/list/src/pages/listCreate.tsx
import * as React from "react";
import { useNavigate } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";

/**
 * ListCreate
 * - 出品の新規作成ページ
 * - 左ペイン: 出品情報入力フォーム
 * - 右ペイン: 管理情報 (AdminCard)
 */
export default function ListCreate() {
  const navigate = useNavigate();

  // ──────────────────────────────────────────────
  // 入力フォーム状態（プリフィルは空）
  // ──────────────────────────────────────────────
  const [product, setProduct] = React.useState("");
  const [brand, setBrand] = React.useState("");
  const [token, setToken] = React.useState("");
  const [stock, setStock] = React.useState<number | "">("");
  const [manager, setManager] = React.useState("");
  const [status, setStatus] = React.useState<"出品中" | "停止中" | "">("");

  // 管理情報
  const [assignee, setAssignee] = React.useState("未設定");
  const [creator] = React.useState("現在のユーザー");
  const [createdAt] = React.useState(new Date().toLocaleDateString());

  // ハンドラ
  const onCreate = () => {
    alert("出品情報を作成しました（ダミー）");
    navigate("/list");
  };

  const onBack = () => navigate(-1);

  return (
    <PageStyle
      layout="grid-2"
      title="出品の作成"
      onBack={onBack}
      onSave={onCreate} // 保存ボタン → 作成ボタンとして利用
    >
      {/* --- 左ペイン（出品作成フォーム） --- */}
      <div className="list-create-form">
        <h2 className="section-title">出品情報</h2>

        <div className="form-group">
          <label>プロダクト名</label>
          <input
            type="text"
            value={product}
            onChange={(e) => setProduct(e.target.value)}
            placeholder="例: シルクブラウス プレミアムライン"
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
          <label>トークン</label>
          <input
            type="text"
            value={token}
            onChange={(e) => setToken(e.target.value)}
            placeholder="例: LUMINA VIP Token"
          />
        </div>

        <div className="form-group">
          <label>在庫数</label>
          <input
            type="number"
            value={stock}
            onChange={(e) =>
              setStock(e.target.value === "" ? "" : Number(e.target.value))
            }
            placeholder="例: 200"
          />
        </div>

        <div className="form-group">
          <label>担当者</label>
          <input
            type="text"
            value={manager}
            onChange={(e) => setManager(e.target.value)}
            placeholder="例: 山田 太郎"
          />
        </div>

        <div className="form-group">
          <label>ステータス</label>
          <select
            value={status}
            onChange={(e) =>
              setStatus(e.target.value as "出品中" | "停止中" | "")
            }
          >
            <option value="">選択してください</option>
            <option value="出品中">出品中</option>
            <option value="停止中">停止中</option>
          </select>
        </div>
      </div>

      {/* --- 右ペイン（管理情報カード） --- */}
      <AdminCard
        title="管理情報"
        assigneeName={assignee}
        onEditAssignee={() => setAssignee("変更済み担当者")}
        onClickAssignee={() => console.log("Assignee clicked:", assignee)}
      />
    </PageStyle>
  );
}
