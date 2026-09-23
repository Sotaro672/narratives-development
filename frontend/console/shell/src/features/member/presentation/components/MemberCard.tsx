// frontend/console/shell/src/features/member/presentation/components/MemberCard.tsx

import * as React from "react";
import {
  Calendar,
  Mail,
  User,
} from "lucide-react";

import {
  Card,
  CardContent,
  CardField,
  CardFields,
  CardHeader,
  CardHeaderIcon,
  CardHeaderLeft,
  CardTitle,
} from "../../../../shared/ui/card";
import Empty from "../../../../shared/ui/empty";
import { ErrorMessage } from "../../../../shared/ui/error";
import Stack from "../../../../shared/ui/stack";
import Text from "../../../../shared/ui/text";

import type { MemberDetail } from "../../application/memberDetailService";

import "../../../../styles/member.css";

const IconUser =
  User as unknown as React.ComponentType<React.SVGProps<SVGSVGElement>>;
const IconMail =
  Mail as unknown as React.ComponentType<React.SVGProps<SVGSVGElement>>;
const IconCalendar =
  Calendar as unknown as React.ComponentType<React.SVGProps<SVGSVGElement>>;

type MemberDetailCardProps = {
  member: MemberDetail | null;
  loading: boolean;
  error: Error | null;
};

function formatDate(iso?: string | null): string {
  if (!iso) return "-";

  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "-";

  return date.toLocaleDateString("ja-JP", {
    year: "numeric",
    month: "long",
    day: "numeric",
  });
}

function MemberCardHeader() {
  return (
    <CardHeader>
      <CardHeaderLeft>
        <CardHeaderIcon>
          <IconUser className="card__header-icon-svg" />
        </CardHeaderIcon>
        <CardTitle strong>基本情報</CardTitle>
      </CardHeaderLeft>
    </CardHeader>
  );
}

export default function MemberDetailCard({
  member,
  loading,
  error,
}: MemberDetailCardProps) {
  if (loading) {
    return (
      <Card>
        <MemberCardHeader />
        <CardContent>
          <Text tone="muted">
            読み込み中です…
          </Text>
        </CardContent>
      </Card>
    );
  }

  if (error) {
    return (
      <Card>
        <MemberCardHeader />
        <CardContent>
          <ErrorMessage>
            データ取得エラー: {error.message}
          </ErrorMessage>
        </CardContent>
      </Card>
    );
  }

  if (!member) {
    return (
      <Card>
        <MemberCardHeader />
        <CardContent>
          <Empty
            compact
            description="該当するメンバーが見つかりません。"
          />
        </CardContent>
      </Card>
    );
  }

  const fullName = [
    member.lastName,
    member.firstName,
  ]
    .filter((value) => value.length > 0)
    .join(" ");

  const fullKana = [
    member.lastNameKana,
    member.firstNameKana,
  ]
    .filter((value) => value.length > 0)
    .join(" ");

  const joinedAt = formatDate(member.createdAt);
  const updatedAt = formatDate(member.updatedAt);

  return (
    <Card>
      <MemberCardHeader />

      <CardContent>
        <Stack gap="lg">
          <CardFields>
            <CardField>
              <Text
                as="div"
                size="xs"
                tone="muted"
                weight="semibold"
              >
                氏名
              </Text>

              <div className="member-card__value">
                <IconUser className="icon-inline" />
                <Text weight="medium">
                  {fullName || "-"}
                </Text>
              </div>
            </CardField>

            <CardField>
              <Text
                as="div"
                size="xs"
                tone="muted"
                weight="semibold"
              >
                読み仮名
              </Text>

              <div className="member-card__value">
                <IconUser className="icon-inline" />
                <Text>{fullKana || "-"}</Text>
              </div>
            </CardField>
          </CardFields>

          <CardFields>
            <CardField>
              <Text
                as="div"
                size="xs"
                tone="muted"
                weight="semibold"
              >
                メールアドレス
              </Text>

              <div className="member-card__value">
                <IconMail className="icon-inline" />
                <Text wrap="anywhere">
                  {member.email}
                </Text>
              </div>
            </CardField>
          </CardFields>

          <CardFields>
            <CardField>
              <Text
                as="div"
                size="xs"
                tone="muted"
                weight="semibold"
              >
                更新日
              </Text>

              <div className="member-card__value">
                <IconCalendar className="icon-inline" />
                <Text>{updatedAt}</Text>
              </div>
            </CardField>

            <CardField>
              <Text
                as="div"
                size="xs"
                tone="muted"
                weight="semibold"
              >
                参加日
              </Text>

              <div className="member-card__value">
                <IconCalendar className="icon-inline" />
                <Text>{joinedAt}</Text>
              </div>
            </CardField>
          </CardFields>
        </Stack>
      </CardContent>
    </Card>
  );
}