// frontend/console/shell/src/pages/memberDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../layout/PageStyle/PageStyle";
import MemberDetailCard from "../features/member/presentation/components/MemberCard";
import { useMemberDetail } from "../features/member/presentation/hooks/useMemberDetail";
import { cancelMemberInvitation } from "../features/member/application/invitationService";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../shared/ui/card";
import { BrandCard } from "../features/member/presentation/components/BrandCard";

export default function MemberDetail() {
  const navigate = useNavigate();
  const { memberId } = useParams<{ memberId: string }>();

  const {
    member,
    memberName,
    assignedBrands,
    brandRows,
    permissions,
    groupedPermissionsByCategory,
    hasGroupedPermissions,
    loading,
    isInvitationPending,
  } = useMemberDetail(memberId);

  const handleBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  const handleDelete = React.useCallback(async () => {
    if (!member || !isInvitationPending) {
      return;
    }

    const confirmed = window.confirm(
      "このメンバーの招待を取り消して削除しますか？送信済みの招待URLも無効になります。",
    );
    if (!confirmed) {
      return;
    }

    try {
      await cancelMemberInvitation(member.id);
      navigate("/member");
    } catch (error: unknown) {
      const message =
        error instanceof Error
          ? error.message
          : "招待の取消に失敗しました。";

      window.alert(message);
    }
  }, [member, isInvitationPending, navigate]);

  if (!memberId) {
    return (
      <PageStyle layout="single" title="メンバー詳細" onBack={handleBack}>
        <div className="p-4 text-red-500">
          メンバーIDが指定されていません。
        </div>
      </PageStyle>
    );
  }

  return (
    <PageStyle
      layout="grid-2"
      title={memberName}
      onBack={handleBack}
      onDelete={!loading && isInvitationPending ? handleDelete : undefined}
    >
      <div>
        <MemberDetailCard memberId={memberId} />
      </div>

      <div className="space-y-4">
        <BrandCard assignedBrands={assignedBrands} brandRows={brandRows} />

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