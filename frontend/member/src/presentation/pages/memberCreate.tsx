// frontend/member/src/presentation/pages/memberCreate.tsx
import * as React from "react";
import { useNavigate } from "react-router-dom";

import { MEMBER_ROLES, type MemberRole, type Member } from "../../domain/entity/member";
import { MemberRepositoryFS } from "../../infrastructure/firestore/memberRepositoryFS";
import type { MemberPatch } from "../../domain/entity/member";

/**
 * シンプルなメンバー作成フォーム
 * - 必須: role
 * - 任意: 氏名 / かな / email / permissions / assignedBrands
 * - id は crypto.randomUUID() で生成
 * - createdAt / updatedAt は ISO8601
 */
export default function MemberCreatePage() {
  const navigate = useNavigate();
  const repo = React.useMemo(() => new MemberRepositoryFS(), []);

  // ---- フォーム状態 ----
  const [firstName, setFirstName] = React.useState("");
  const [lastName, setLastName] = React.useState("");
  const [firstNameKana, setFirstNameKana] = React.useState("");
  const [lastNameKana, setLastNameKana] = React.useState("");
  const [email, setEmail] = React.useState("");
  const [role, setRole] = React.useState<MemberRole>("brand-manager");
  const [permissionsText, setPermissionsText] = React.useState(""); // カンマ区切り
  const [brandsText, setBrandsText] = React.useState(""); // カンマ区切り

  const [submitting, setSubmitting] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  const toArray = (s: string) =>
    s
      .split(",")
      .map((x) => x.trim())
      .filter(Boolean);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      const id = crypto.randomUUID();
      const now = new Date().toISOString();

      const member: Member = {
        id,
        firstName: firstName.trim() || undefined,
        lastName: lastName.trim() || undefined,
        firstNameKana: firstNameKana.trim() || undefined,
        lastNameKana: lastNameKana.trim() || undefined,
        email: email.trim() || undefined,
        role,
        permissions: toArray(permissionsText),
        assignedBrands: (() => {
          const arr = toArray(brandsText);
          return arr.length ? arr : undefined;
        })(),
        createdAt: now,
        updatedAt: now,
        updatedBy: "console", // 必要に応じてログインユーザーIDへ
        deletedAt: null,
        deletedBy: null,
      };

      await repo.create(member);
      // 作成後は一覧へ戻る
      navigate("/member");
    } catch (err: any) {
      setError(err?.message ?? String(err));
    } finally {
      setSubmitting(false);
    }
  };

  const cancel = () => navigate(-1);

  return (
    <div className="p-4 max-w-3xl mx-auto">
      <h1 className="text-xl font-semibold mb-4">メンバー追加</h1>

      {error && <div className="mb-3 text-red-500">エラー: {error}</div>}

      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="block text-sm text-slate-300 mb-1">姓</label>
            <input
              className="w-full rounded border border-slate-600 bg-slate-800 px-3 py-2"
              value={lastName}
              onChange={(e) => setLastName(e.target.value)}
              placeholder="山田"
            />
          </div>
          <div>
            <label className="block text-sm text-slate-300 mb-1">名</label>
            <input
              className="w-full rounded border border-slate-600 bg-slate-800 px-3 py-2"
              value={firstName}
              onChange={(e) => setFirstName(e.target.value)}
              placeholder="太郎"
            />
          </div>
          <div>
            <label className="block text-sm text-slate-300 mb-1">姓（かな）</label>
            <input
              className="w-full rounded border border-slate-600 bg-slate-800 px-3 py-2"
              value={lastNameKana}
              onChange={(e) => setLastNameKana(e.target.value)}
              placeholder="やまだ"
            />
          </div>
          <div>
            <label className="block text-sm text-slate-300 mb-1">名（かな）</label>
            <input
              className="w-full rounded border border-slate-600 bg-slate-800 px-3 py-2"
              value={firstNameKana}
              onChange={(e) => setFirstNameKana(e.target.value)}
              placeholder="たろう"
            />
          </div>
        </div>

        <div>
          <label className="block text-sm text-slate-300 mb-1">メールアドレス（任意）</label>
          <input
            type="email"
            className="w-full rounded border border-slate-600 bg-slate-800 px-3 py-2"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="taro@example.com"
          />
        </div>

        <div>
          <label className="block text-sm text-slate-300 mb-1">ロール（必須）</label>
          <select
            className="w-full rounded border border-slate-600 bg-slate-800 px-3 py-2"
            value={role}
            onChange={(e) => setRole(e.target.value as MemberRole)}
            required
          >
            {MEMBER_ROLES.map((r) => (
              <option key={r} value={r}>
                {r}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label className="block text-sm text-slate-300 mb-1">
            権限名（カンマ区切り）
          </label>
          <input
            className="w-full rounded border border-slate-600 bg-slate-800 px-3 py-2"
            value={permissionsText}
            onChange={(e) => setPermissionsText(e.target.value)}
            placeholder="member.read, member.write"
          />
        </div>

        <div>
          <label className="block text-sm text-slate-300 mb-1">
            割当ブランドID（カンマ区切り）
          </label>
          <input
            className="w-full rounded border border-slate-600 bg-slate-800 px-3 py-2"
            value={brandsText}
            onChange={(e) => setBrandsText(e.target.value)}
            placeholder="LUMINA, NEXUS"
          />
        </div>

        <div className="flex gap-3 pt-2">
          <button
            type="submit"
            disabled={submitting}
            className="rounded bg-blue-600 hover:bg-blue-500 px-4 py-2 disabled:opacity-60"
          >
            {submitting ? "作成中..." : "作成する"}
          </button>
          <button
            type="button"
            onClick={cancel}
            className="rounded bg-slate-700 hover:bg-slate-600 px-4 py-2"
          >
            キャンセル
          </button>
        </div>
      </form>
    </div>
  );
}
