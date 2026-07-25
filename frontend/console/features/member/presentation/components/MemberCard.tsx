// frontend/member/src/presentation/components/MemberDetailCard.tsx

import * as React from "react";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shell/src/shared/ui/card";
import { User, Mail, Calendar } from "lucide-react";
import { useMemberDetail } from "../hooks/useMemberDetail";

const IconUser = User as unknown as React.ComponentType<
  React.SVGProps<SVGSVGElement>
>;
const IconMail = Mail as unknown as React.ComponentType<
  React.SVGProps<SVGSVGElement>
>;
const IconCalendar = Calendar as unknown as React.ComponentType<
  React.SVGProps<SVGSVGElement>
>;

type MemberDetailCardProps = {
  /**
   * 互換のため prop 名は memberId のままにする。
   *
   * IMPORTANT:
   * この値には Firestore member docId ではなく Firebase Auth UID を渡す。
   *
   * backend:
   * - GET /members/{uid} は Firebase UID 専用
   * - PATCH /members/{docId} は Firestore member docId 専用
   */
  memberId: string;
};

function formatDate(iso?: string | null): string {
  if (!iso) return "-";

  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "-";

  return d.toLocaleDateString("ja-JP", {
    year: "numeric",
    month: "long",
    day: "numeric",
  });
}

export default function MemberDetailCard({ memberId }: MemberDetailCardProps) {
  const memberUid = String(memberId ?? "").trim();

  const { member, loading, error } = useMemberDetail(memberUid);

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

  const fullName = `${member.lastName ?? ""} ${member.firstName ?? ""}`.trim();
  const fullKana = `${member.lastNameKana ?? ""} ${member.firstNameKana ?? ""}`.trim();

  const email = member.email || "-";
  const joinedAt = formatDate(member.createdAt);
  const updatedAt = formatDate(member.updatedAt || member.createdAt);

  return (
    <Card className="member-card w-full">
      <CardHeader className="member-card__header">
        <CardTitle className="member-card__title flex items-center gap-2">
          <IconUser className="member-card__icon w-4 h-4" />
          基本情報
        </CardTitle>
      </CardHeader>

      <CardContent className="member-card__body space-y-6 text-sm">
        {/* 氏名・読み仮名 */}
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

        {/* メールアドレス */}
        <div className="member-card__grid">
          <div className="member-card__section">
            <div className="member-card__label">メールアドレス</div>
            <div className="member-card__value">
              <IconMail className="icon-inline w-4 h-4" />
              <span className="break-all">{email}</span>
            </div>
          </div>
        </div>

        {/* 更新日・参加日 */}
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