// frontend/member/src/presentation/pages/memberCreate.tsx
import * as React from "react";
import { useNavigate } from "react-router-dom";
import { MEMBER_ROLES, type MemberRole } from "../../domain/entity/member";
import { useMemberCreate } from "../../hooks/useMemberCreate";

export default function MemberCreatePage() {
  const navigate = useNavigate();

  const {
    firstName, lastName, firstNameKana, lastNameKana, email, role,
    permissionsText, brandsText,
    submitting, error,
    setFirstName, setLastName, setFirstNameKana, setLastNameKana,
    setEmail, setRole, setPermissionsText, setBrandsText,
    handleSubmit,
  } = useMemberCreate({
    onSuccess: () => navigate("/member"),
  });

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
