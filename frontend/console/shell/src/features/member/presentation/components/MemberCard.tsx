// frontend/console/shell/src/features/member/presentation/components/MemberCard.tsx

import * as React from "react";
import { Calendar, Mail, User } from "lucide-react";

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

import type { MemberDetail } from "../../application/memberDetailService";

const IconUser = User as unknown as React.ComponentType<React.SVGProps<SVGSVGElement>>;
const IconMail = Mail as unknown as React.ComponentType<React.SVGProps<SVGSVGElement>>;
const IconCalendar = Calendar as unknown as React.ComponentType<React.SVGProps<SVGSVGElement>>;

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
        <CardContent className="text-sm text-[hsl(var(--muted-foreground))]">
          読み込み中です…
        </CardContent>
      </Card>
    );
  }

  if (error) {
    return (
      <Card>
        <MemberCardHeader />
        <CardContent className="text-sm text-red-500">
          データ取得エラー: {error.message}
        </CardContent>
      </Card>
    );
  }

  if (!member) {
    return (
      <Card>
        <MemberCardHeader />
        <CardContent className="text-sm text-[hsl(var(--muted-foreground))]">
          該当するメンバーが見つかりません。
        </CardContent>
      </Card>
    );
  }

  const fullName = [member.lastName, member.firstName]
    .filter((value) => value.length > 0)
    .join(" ");

  const fullKana = [member.lastNameKana, member.firstNameKana]
    .filter((value) => value.length > 0)
    .join(" ");

  const joinedAt = formatDate(member.createdAt);
  const updatedAt = formatDate(member.updatedAt);

  return (
    <Card>
      <MemberCardHeader />

      <CardContent className="space-y-6 text-sm">
        <CardFields className="member-card__grid">
          <CardField className="member-card__section">
            <div className="member-card__label">氏名</div>
            <div className="member-card__value">
              <IconUser className="icon-inline w-4 h-4" />
              <span className="font-medium">{fullName || "-"}</span>
            </div>
          </CardField>

          <CardField className="member-card__section">
            <div className="member-card__label">読み仮名</div>
            <div className="member-card__value">
              <IconUser className="icon-inline w-4 h-4" />
              <span>{fullKana || "-"}</span>
            </div>
          </CardField>
        </CardFields>

        <CardFields className="member-card__grid">
          <CardField className="member-card__section">
            <div className="member-card__label">メールアドレス</div>
            <div className="member-card__value">
              <IconMail className="icon-inline w-4 h-4" />
              <span className="break-all">{member.email}</span>
            </div>
          </CardField>
        </CardFields>

        <CardFields className="member-card__grid">
          <CardField className="member-card__section">
            <div className="member-card__label">更新日</div>
            <div className="member-card__value">
              <IconCalendar className="icon-inline w-4 h-4" />
              <span>{updatedAt}</span>
            </div>
          </CardField>

          <CardField className="member-card__section">
            <div className="member-card__label">参加日</div>
            <div className="member-card__value">
              <IconCalendar className="icon-inline w-4 h-4" />
              <span>{joinedAt}</span>
            </div>
          </CardField>
        </CardFields>
      </CardContent>
    </Card>
  );
}