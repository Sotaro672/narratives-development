// frontend/member/src/presentation/pages/memberDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import MemberDetailCard from "../components/MemberCard";
import { useMemberDetail } from "../hooks/useMemberDetail";
import { BrandCard } from "../components/BrandCard";
import { PermissionCard } from "../components/PermissionCard";

export default function MemberDetail() {
  const navigate = useNavigate();
  const { memberId } = useParams<{ memberId: string }>();

  // Firestore から詳細取得（招待中判定含む）
  const { memberName, assignedBrands, permissions } = useMemberDetail(memberId);

  const handleBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  return (
    <PageStyle
      layout="grid-2"
      title={`メンバー詳細：${memberName}`}
      onBack={handleBack}
    >
      {/* 左カラム：基本情報カード */}
      <div>
        <MemberDetailCard memberId={memberId ?? ""} />
      </div>

      {/* 右カラム：所属ブランドカード + 権限カード */}
      <div className="space-y-4">
        <BrandCard assignedBrands={assignedBrands} />
        <PermissionCard permissions={permissions} />
      </div>
    </PageStyle>
  );
}
