// frontend/member/src/pages/memberDetail.tsx
import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../shell/src/layout/PageStyle/PageStyle";

/**
 * メンバー詳細ページ
 * - 左ペイン：メンバー情報
 */
export default function MemberDetail() {
  const navigate = useNavigate();
  const { memberId } = useParams<{ memberId: string }>();

  // ──────────────────────────────────────────────
  // モックデータ（API連携前の仮データ）
  // ──────────────────────────────────────────────
  const [member] = React.useState({
    name: "佐藤 美咲",
    email: "misaki.sato@example.com",
    role: "ブランドマネージャー",
    organization: "LUMINA Fashion",
    joinedAt: "2024/04/10",
    status: "アクティブ",
  });

  // 戻るボタン動作
  const handleBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  return (
    <PageStyle
      layout="single"
      title={`メンバー詳細：${memberId ?? "不明ID"}`}
      onBack={handleBack}
    >
      {/* ─────────── メンバー情報 ─────────── */}
      <div className="member-detail">
        <h2 className="member-detail__title">メンバー情報</h2>
        <table className="member-detail__table">
          <tbody>
            <tr>
              <th>氏名</th>
              <td>{member.name}</td>
            </tr>
            <tr>
              <th>メールアドレス</th>
              <td>{member.email}</td>
            </tr>
            <tr>
              <th>役職 / 権限</th>
              <td>{member.role}</td>
            </tr>
            <tr>
              <th>所属組織</th>
              <td>{member.organization}</td>
            </tr>
            <tr>
              <th>登録日</th>
              <td>{member.joinedAt}</td>
            </tr>
            <tr>
              <th>ステータス</th>
              <td>
                <span
                  className={`status-badge ${
                    member.status === "アクティブ" ? "active" : "inactive"
                  }`}
                >
                  {member.status}
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </PageStyle>
  );
}
