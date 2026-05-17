// frontend/member/src/presentation/pages/memberDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import MemberDetailCard from "../components/MemberCard";
import { useMemberDetail } from "../hooks/useMemberDetail";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shell/src/shared/ui/card";
import { BrandCard } from "../components/BrandCard";

export default function MemberDetail() {
  const navigate = useNavigate();

  /**
   * IMPORTANT:
   * この route param は Firestore member docId ではなく Firebase Auth UID。
   *
   * backend:
   * - GET /members/{uid} は Firebase UID 専用
   * - PATCH /members/{docId} は Firestore member docId 専用
   */
  const { memberUid } = useParams<{ memberUid: string }>();

  const {
    memberName,
    assignedBrands,
    brandRows,
    permissions,
    groupedPermissionsByCategory,
    hasGroupedPermissions,
  } = useMemberDetail(memberUid);

  const handleBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  return (
    <PageStyle layout="grid-2" title={`${memberName}`} onBack={handleBack}>
      {/* 左カラム：基本情報カード */}
      <div>
        {/*
          MemberDetailCard は memberId prop だけを受け取る。
          ただし現状方針では、この memberId には Firebase Auth UID を渡す。
        */}
        <MemberDetailCard memberId={memberUid ?? ""} />
      </div>

      {/* 右カラム：所属ブランドカード + 権限カード */}
      <div className="space-y-4">
        {/* 所属ブランド */}
        <BrandCard assignedBrands={assignedBrands} brandRows={brandRows} />

        {/* 権限カード */}
        <Card>
          <CardHeader>
            <CardTitle>権限</CardTitle>
          </CardHeader>
          <CardContent>
            {permissions.length === 0 ? (
              <p className="text-sm text-[hsl(var(--muted-foreground))]">
                権限は未設定です。
              </p>
            ) : !hasGroupedPermissions ? (
              <p className="text-sm text-[hsl(var(--muted-foreground))]">
                権限情報を読み込み中です…
              </p>
            ) : (
              <div className="space-y-3">
                {Object.entries(groupedPermissionsByCategory).map(
                  ([category, perms]) => (
                    <div key={category}>
                      <div className="text-xs font-semibold text-slate-500 mb-1">
                        {category}
                      </div>

                      <ul className="text-sm space-y-1 ml-3 list-disc">
                        {perms?.map((perm: string) => (
                          <li key={`${category}:${perm}`}>{perm}</li>
                        ))}
                      </ul>
                    </div>
                  ),
                )}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </PageStyle>
  );
}