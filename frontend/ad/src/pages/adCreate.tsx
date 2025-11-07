// frontend/ad/src/pages/adCreate.tsx
import * as React from "react";
import { useNavigate } from "react-router-dom";
import PageStyle from "../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../admin/src/pages/AdminCard";

/**
 * 広告キャンペーン作成ページ
 * - 左ペイン: 広告情報入力フォーム
 * - 右ペイン: 管理情報 (AdminCard)
 */
export default function AdCreate() {
  const navigate = useNavigate();

  // ──────────────────────────────────────────────
  // 入力フォーム状態
  // ──────────────────────────────────────────────
  const [campaign, setCampaign] = React.useState("");
  const [brand, setBrand] = React.useState("");
  const [owner, setOwner] = React.useState("");
  const [startDate, setStartDate] = React.useState("");
  const [endDate, setEndDate] = React.useState("");
  const [budget, setBudget] = React.useState<number | "">("");
  const [status, setStatus] = React.useState<"実行中" | "停止中" | "">("");

  // 管理情報
  const [assignee, setAssignee] = React.useState("未設定");
  const [creator] = React.useState("現在のユーザー");
  const [createdAt] = React.useState(new Date().toLocaleDateString());

  // 作成処理
  const onCreate = () => {
    alert("新しい広告キャンペーンを作成しました（ダミー）");
    navigate("/ad");
  };

  const onBack = () => navigate(-1);

  return (
    <PageStyle
      layout="grid-2"
      title="キャンペーンの作成"
      onBack={onBack}
      onSave={onCreate} // 保存ボタン→作成ボタンとして使用
    >
      {/* ───────── 左カラム（フォーム）───────── */}
      <div className="ad-create-form">
        <h2 className="section-title">キャンペーン情報</h2>

        <div className="form-group">
          <label>キャンペーン名</label>
          <input
            type="text"
            value={campaign}
            onChange={(e) => setCampaign(e.target.value)}
            placeholder="例: LUMINA 春コレクション"
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
            value={owner}
            onChange={(e) => setOwner(e.target.value)}
            placeholder="例: 佐藤 美咲"
          />
        </div>

        <div className="form-row">
          <div className="form-group half">
            <label>開始日</label>
            <input
              type="date"
              value={startDate}
              onChange={(e) => setStartDate(e.target.value)}
            />
          </div>
          <div className="form-group half">
            <label>終了日</label>
            <input
              type="date"
              value={endDate}
              onChange={(e) => setEndDate(e.target.value)}
            />
          </div>
        </div>

        <div className="form-group">
          <label>予算（円）</label>
          <input
            type="number"
            value={budget}
            onChange={(e) =>
              setBudget(e.target.value === "" ? "" : Number(e.target.value))
            }
            placeholder="例: 500000"
          />
        </div>

        <div className="form-group">
          <label>ステータス</label>
          <select
            value={status}
            onChange={(e) =>
              setStatus(e.target.value as "実行中" | "停止中" | "")
            }
          >
            <option value="">選択してください</option>
            <option value="実行中">実行中</option>
            <option value="停止中">停止中</option>
          </select>
        </div>
      </div>

      {/* ───────── 右カラム（管理情報カード）───────── */}
      <AdminCard
        title="管理情報"
        assigneeName={assignee}
        createdByName={creator}
        createdAt={createdAt}
        onEditAssignee={() => setAssignee("変更済み担当者")}
        onClickAssignee={() => console.log("assignee clicked:", assignee)}
        onClickCreatedBy={() => console.log("createdBy clicked:", creator)}
      />
    </PageStyle>
  );
}
