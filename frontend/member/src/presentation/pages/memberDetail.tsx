// frontend/member/src/presentation/pages/memberDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import MemberDetailCard from "../components/MemberDetailCard";
import { MOCK_MEMBERS } from "../../infrastructure/mock/member_mockdata";

export default function MemberDetail() {
  const navigate = useNavigate();
  const { memberId } = useParams<{ memberId: string }>();

  // ─────────────────────────────────────────────
  // モックデータから対象メンバーを検索（PageHeader 表示用）
  // ─────────────────────────────────────────────
  const member = React.useMemo(() => {
    if (!memberId) return undefined;
    return MOCK_MEMBERS.find((m) => m.id === memberId);
  }, [memberId]);

  const memberName = React.useMemo(() => {
    if (!member) return "不明なメンバー";
    const fullName = [member.lastName, member.firstName]
      .filter(Boolean)
      .join(" ");
    return fullName || "不明なメンバー";
  }, [member]);

  const handleBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  return (
    <PageStyle
      layout="grid-2"
      title={`メンバー詳細：${memberName}`}
      onBack={handleBack}
    >
      {/* 左カラム：基本情報カード（memberId を渡して内部で取得） */}
      <div>
        <MemberDetailCard memberId={memberId ?? ""} />
      </div>

      {/* 右カラム：将来拡張用プレースホルダー */}
      <div></div>
    </PageStyle>
  );
}
