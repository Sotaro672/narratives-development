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
  memberId: string;
};

/**
 * ISO8601 文字列を日本語の日付表記に変換
 */
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

/**
 * メンバー詳細カード
 * - memberId から Firestore 経由でデータを取得し、各項目を表示
 * - 氏名が null/空文字の場合は「氏名」欄を空欄にする（ID で埋めない）
 */
export default function MemberDetailCard({ memberId }: MemberDetailCardProps) {
  const { member, loading, error } = useMemberDetail(memberId);

  // ローディング表示
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

  // エラー表示
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

  // メンバーが存在しない場合
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

  // 氏名と読み仮名の組み立て
  const rawFullName = `${member.lastName ?? ""} ${member.firstName ?? ""}`.trim();
  // ★ 氏名が null/空の場合は「空欄」にする（"-" や ID では埋めない）
  const fullName = rawFullName || "";

  const rawFullKana = `${member.lastNameKana ?? ""} ${
    member.firstNameKana ?? ""
  }`.trim();
  const fullKana = rawFullKana || "";

  const email = member.email || "-";
  const joinedAt = formatDate(member.createdAt);
  const updatedAt = formatDate(member.updatedAt || member.createdAt);

  // 氏名が設定されているか
  const hasName = fullName !== "";

  // Header のタイトル
  // ★ name が null/空の場合は ID を表示しない（空欄扱い）
  const headerTitle = hasName
    ? `基本情報（ID: ${memberId}）`
    : "基本情報";

  // ─────────────────────────────────────────────
  // 描画
  // ─────────────────────────────────────────────
  return (
    <Card className="member-card w-full">
      {/* Header */}
      <CardHeader className="member-card__header">
        <CardTitle className="member-card__title flex items-center gap-2">
          <IconUser className="member-card__icon w-4 h-4" />
          {headerTitle}
        </CardTitle>
      </CardHeader>

      {/* Content */}
      <CardContent className="member-card__body space-y-6 text-sm">
        {/* 氏名・読み仮名 */}
        <div className="member-card__grid">
          <div className="member-card__section">
            <div className="member-card__label">氏名</div>
            <div className="member-card__value">
              <IconUser className="icon-inline w-4 h-4" />
              <span className="font-medium">{fullName}</span>
            </div>
          </div>

          <div className="member-card__section">
            <div className="member-card__label">読み仮名</div>
            <div className="member-card__value">
              <IconUser className="icon-inline w-4 h-4" />
              <span>{fullKana}</span>
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
