// frontend/ad/src/pages/adDetail.tsx
import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";

/**
 * 広告キャンペーン詳細ページ
 * - 左ペイン：キャンペーン情報表示
 * - 右ペイン：管理情報カード
 */
export default function AdDetail() {
  const navigate = useNavigate();
  const { adId } = useParams<{ adId: string }>();

  // ──────────────────────────────────────────────
  // モックデータ
  // ──────────────────────────────────────────────
  const [campaign] = React.useState("LUMINA 春コレクション");
  const [brand] = React.useState("LUMINA Fashion");
  const [owner] = React.useState("佐藤 美咲");
  const [period] = React.useState("2024/3/1 - 2024/3/31");
  const [status] = React.useState("実行中");
  const [spendRate] = React.useState("68.4%");
  const [spend] = React.useState("¥342,000");
  const [budget] = React.useState("¥500,000");

  // 管理情報
  const [assignee, setAssignee] = React.useState("佐藤 美咲");
  const [creator] = React.useState("渡辺 花子");
  const [createdAt] = React.useState("2024/2/15");

  // 戻るボタン
  const onBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  return (
    <PageStyle
      layout="grid-2"
      title={`キャンペーン詳細：${adId ?? "不明ID"}`}
      onBack={onBack}
    >
      {/* 左ペイン：キャンペーン情報 */}
      <div className="ad-detail">
        <h2 className="ad-detail__title">キャンペーン情報</h2>
        <table className="ad-detail__table">
          <tbody>
            <tr>
              <th>キャンペーン名</th>
              <td>{campaign}</td>
            </tr>
            <tr>
              <th>ブランド</th>
              <td>{brand}</td>
            </tr>
            <tr>
              <th>担当者</th>
              <td>{owner}</td>
            </tr>
            <tr>
              <th>広告期間</th>
              <td>{period}</td>
            </tr>
            <tr>
              <th>ステータス</th>
              <td>
                <span className="ad-status-badge">{status}</span>
              </td>
            </tr>
            <tr>
              <th>消化率</th>
              <td>{spendRate}</td>
            </tr>
            <tr>
              <th>消化金額</th>
              <td>{spend}</td>
            </tr>
            <tr>
              <th>予算</th>
              <td>{budget}</td>
            </tr>
          </tbody>
        </table>
      </div>

      {/* 右ペイン：管理情報 */}
      <AdminCard
        title="管理情報"
        assigneeName={assignee}
        createdByName={creator}
        createdAt={createdAt}
        onEditAssignee={() => setAssignee("新担当者")}
        onClickAssignee={() => console.log("assignee clicked:", assignee)}
        onClickCreatedBy={() => console.log("createdBy clicked:", creator)}
      />
    </PageStyle>
  );
}
