// frontend/console/shell/src/features/member/presentation/components/MemberCard.tsx

import * as React from "react";
import { Card, CardContent, CardHeader, CardTitle } from "../../../../shared/ui/card";
import { Calendar, Mail, User } from "lucide-react";
import { useMemberDetail } from "../hooks/useMemberDetail";

const IconUser = User as unknown as React.ComponentType<React.SVGProps<SVGSVGElement>>;
const IconMail = Mail as unknown as React.ComponentType<React.SVGProps<SVGSVGElement>>;
const IconCalendar = Calendar as unknown as React.ComponentType<React.SVGProps<SVGSVGElement>>;

type MemberDetailCardProps = {
  /**
   * Firestore Member document ID。
   *
   * Backend:
   * GET /members/by-id/{memberId}
   */
  memberId: string;
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

export default function MemberDetailCard({ memberId }: MemberDetailCardProps) {
  const { member, loading, error } = useMemberDetail(memberId);

  if (loading) {
    return (
      <Card className="member-card w-full">
        <CardHeader className="member-card__header">
          <CardTitle className="member-card__title flex items-center gap-2">
            <IconUser className="member-card__icon w-4 h-4" />
            基本情報
          </CardTitle>
        </CardHeader>
        <CardContent className="p-6 text-sm text-[hsl(var(--muted-foreground))]">
          読み込み中です…
        </CardContent>
      </Card>
    );
  }

  if (error) {
    return (
      <Card className="member-card w-full">
        <CardHeader className="member-card__header">
          <CardTitle className="member-card__title flex items-center gap-2">
            <IconUser className="member-card__icon w-4 h-4" />
            基本情報
          </CardTitle>
        </CardHeader>
        <CardContent className="p-6 text-sm text-red-500">
          データ取得エラー: {error.message}
        </CardContent>
      </Card>
    );
  }

  if (!member) {
    return (
      <Card className="member-card w-full">
        <CardHeader className="member-card__header">
          <CardTitle className="member-card__title flex items-center gap-2">
            <IconUser className="member-card__icon w-4 h-4" />
            基本情報
          </CardTitle>
        </CardHeader>
        <CardContent className="p-6 text-sm text-[hsl(var(--muted-foreground))]">
          該当するメンバーが見つかりません。
        </CardContent>
      </Card>
    );
  }

  const fullName = [member.lastName, member.firstName].filter((value) => value.length > 0).join(" ");
  const fullKana = [member.lastNameKana, member.firstNameKana].filter((value) => value.length > 0).join(" ");
  const joinedAt = formatDate(member.createdAt);
  const updatedAt = formatDate(member.updatedAt);

  return (
    <Card className="member-card w-full">
      <CardHeader className="member-card__header">
        <CardTitle className="member-card__title flex items-center gap-2">
          <IconUser className="member-card__icon w-4 h-4" />
          基本情報
        </CardTitle>
      </CardHeader>

      <CardContent className="member-card__body space-y-6 text-sm">
        <div className="member-card__grid">
          <div className="member-card__section">
            <div className="member-card__label">氏名</div>
            <div className="member-card__value">
              <IconUser className="icon-inline w-4 h-4" />
              <span className="font-medium">{fullName || "-"}</span>
            </div>
          </div>

          <div className="member-card__section">
            <div className="member-card__label">読み仮名</div>
            <div className="member-card__value">
              <IconUser className="icon-inline w-4 h-4" />
              <span>{fullKana || "-"}</span>
            </div>
          </div>
        </div>

        <div className="member-card__grid">
          <div className="member-card__section">
            <div className="member-card__label">メールアドレス</div>
            <div className="member-card__value">
              <IconMail className="icon-inline w-4 h-4" />
              <span className="break-all">{member.email}</span>
            </div>
          </div>
        </div>

        <div className="member-card__grid">
          <div className="member-card__section">
            <div className="member-card__label">更新日</div>
            <div className="member-card__value">
              <IconCalendar className="icon-inline w-4 h-4" />
              <span>{updatedAt}</span>
            </div>
          </div>

          <div className="member-card__section">
            <div className="member-card__label">参加日</div>
            <div className="member-card__value">
              <IconCalendar className="icon-inline w-4 h-4" />
              <span>{joinedAt}</span>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}