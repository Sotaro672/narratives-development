// frontend/console/shell/src/pages/memberDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

import { cancelMemberInvitation } from "../features/member/application/invitationService";
import { BrandCard } from "../features/member/presentation/components/BrandCard";
import MemberDetailCard from "../features/member/presentation/components/MemberCard";
import { useMemberDetail } from "../features/member/presentation/hooks/useMemberDetail";
import PageStyle from "../layout/PageStyle/PageStyle";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../shared/ui/card";
import { ErrorMessage } from "../shared/ui/error";
import Loading from "../shared/ui/loading";
import Stack from "../shared/ui/stack";
import Text from "../shared/ui/text";

import "../styles/member.css";

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
    error,
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
    } catch (deleteError: unknown) {
      const message =
        deleteError instanceof Error
          ? deleteError.message
          : "招待の取消に失敗しました。";

      window.alert(message);
    }
  }, [member, isInvitationPending, navigate]);

  if (!memberId) {
    return (
      <PageStyle
        layout="single"
        title="メンバー詳細"
        onBack={handleBack}
      >
        <ErrorMessage variant="panel">
          メンバーIDが指定されていません。
        </ErrorMessage>
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
        <MemberDetailCard
          member={member}
          loading={loading}
          error={error}
        />
      </div>

      <div className="page-column">
        <BrandCard
          assignedBrands={assignedBrands}
          brandRows={brandRows}
        />

        <Card>
          <CardHeader>
            <CardTitle>権限</CardTitle>
          </CardHeader>

          <CardContent>
            {loading ? (
              <Loading
                variant="card"
                message="権限情報を読み込み中です..."
              />
            ) : permissions.length === 0 ? (
              <Text as="p" tone="muted">
                権限は未設定です。
              </Text>
            ) : !hasGroupedPermissions ? (
              <Text as="p" tone="muted">
                権限情報を表示できません。
              </Text>
            ) : (
              <Stack gap="md">
                {Object.entries(groupedPermissionsByCategory).map(
                  ([category, perms]) => (
                    <Stack key={category} gap="xs">
                      <Text
                        as="div"
                        size="xs"
                        tone="muted"
                        weight="semibold"
                      >
                        {category}
                      </Text>

                      <ul className="member-permissions__list">
                        {perms?.map((perm: string) => (
                          <li key={`${category}:${perm}`}>
                            <Text>{perm}</Text>
                          </li>
                        ))}
                      </ul>
                    </Stack>
                  ),
                )}
              </Stack>
            )}
          </CardContent>
        </Card>
      </div>
    </PageStyle>
  );
}