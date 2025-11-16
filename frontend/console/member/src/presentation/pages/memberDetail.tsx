// frontend/member/src/presentation/pages/memberDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import MemberDetailCard from "../components/MemberCard";
import { useMemberDetail } from "../hooks/useMemberDetail";

export default function MemberDetail() {
  const navigate = useNavigate();
  const { memberId } = useParams<{ memberId: string }>();

  // Firestore から詳細取得（招待中判定含む）
  const { memberName } = useMemberDetail(memberId);

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

      {/* 右カラム：将来拡張用 */}
      <div></div>
    </PageStyle>
  );
}
