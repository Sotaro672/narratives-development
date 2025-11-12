// frontend/member/src/presentation/pages/memberCreate.tsx
import * as React from "react";
import { useNavigate } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import { MEMBER_ROLES, type MemberRole } from "../../domain/entity/member";
import { useMemberCreate } from "../../hooks/useMemberCreate";
import { Input } from "../../../../shell/src/shared/ui/input";
import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from "../../../../shell/src/shared/ui/popover";
import { Button } from "../../../../shell/src/shared/ui/button";
import "../styles/member.css";

export default function MemberCreatePage() {
  const navigate = useNavigate();
  const formRef = React.useRef<HTMLFormElement>(null);

  const {
    firstName, lastName, firstNameKana, lastNameKana, email, role,
    permissionsText, brandsText,
    submitting, error,
    setFirstName, setLastName, setFirstNameKana, setLastNameKana,
    setEmail, setRole, setPermissionsText, setBrandsText,
    handleSubmit,
    permissionCategories, // ← ★ useMemberCreateから取得（カテゴリ情報）
  } = useMemberCreate({
    onSuccess: () => navigate("/member"),
  });

  const handleBack = () => navigate(-1);
  const handleCreate = () => formRef.current?.requestSubmit();

  const roleLabel = (r?: string) => r ?? "";
  const closePopover = () =>
    document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape" }));

  return (
    <PageStyle title="メンバー追加" onBack={handleBack} onCreate={handleCreate}>
      <div className="p-4 max-w-3xl mx-auto">
        {error && <div className="mb-3 text-red-500">エラー: {error}</div>}

        <form ref={formRef} onSubmit={handleSubmit} className="space-y-4" noValidate>
          {/* 姓 → 姓（かな） */}
          <div className="name-row">
            <div className="name-field">
              <label className="block text-sm text-slate-300 mb-1">姓</label>
              <Input
                variant="default"
                className="w-full rounded border border-slate-600 bg-slate-800 px-3 py-2"
                value={lastName}
                onChange={(e) => setLastName(e.target.value)}
                placeholder="山田"
              />
            </div>
            <div className="name-field">
              <label className="block text-sm text-slate-300 mb-1">姓（かな）</label>
              <Input
                variant="default"
                className="w-full rounded border border-slate-600 bg-slate-800 px-3 py-2"
                value={lastNameKana}
                onChange={(e) => setLastNameKana(e.target.value)}
                placeholder="やまだ"
              />
            </div>
          </div>

          {/* 名 → 名（かな） */}
          <div className="name-row">
            <div className="name-field">
              <label className="block text-sm text-slate-300 mb-1">名</label>
              <Input
                variant="default"
                className="w-full rounded border border-slate-600 bg-slate-800 px-3 py-2"
                value={firstName}
                onChange={(e) => setFirstName(e.target.value)}
                placeholder="太郎"
              />
            </div>
            <div className="name-field">
              <label className="block text-sm text-slate-300 mb-1">名（かな）</label>
              <Input
                variant="default"
                className="w-full rounded border border-slate-600 bg-slate-800 px-3 py-2"
                value={firstNameKana}
                onChange={(e) => setFirstNameKana(e.target.value)}
                placeholder="たろう"
              />
            </div>
          </div>

          {/* メールアドレス（必須） */}
          <div>
            <label className="block text-sm text-slate-300 mb-1">メールアドレス（必須）</label>
            <Input
              type="email"
              required
              autoComplete="email"
              variant="default"
              className="w-full rounded border border-slate-600 bg-slate-800 px-3 py-2"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="taro@example.com"
            />
          </div>

          {/* ロール選択（カテゴリ表示付きPopover） */}
          <div>
            <label className="block text-sm text-slate-300 mb-1">ロール（必須）</label>

            <Popover>
              <PopoverTrigger>
                <Button
                  type="button"
                  variant="outline"
                  className="w-full justify-start border-slate-600 bg-slate-800 text-left"
                >
                  {role ? (
                    roleLabel(role)
                  ) : (
                    <span className="text-slate-400">ロールを選択</span>
                  )}
                </Button>
              </PopoverTrigger>

              {/* PopoverContentにカテゴリを反映 */}
              <PopoverContent className="text-sm max-h-[320px] overflow-y-auto">
                {permissionCategories.map((cat) => (
                  <div key={cat.key} className="mb-3">
                    <div className="text-xs font-semibold text-slate-500 px-2 py-1 border-b border-slate-200">
                      {cat.key}{" "}
                      <span className="text-slate-400">({cat.count})</span>
                    </div>
                    <div className="flex flex-col mt-1">
                      {cat.permissions.map((perm) => (
                        <div
                          key={perm.id}
                          className="px-3 py-1.5 text-slate-700 hover:bg-slate-100 rounded"
                        >
                          <span className="text-xs font-medium">{perm.name}</span>
                          <p className="text-[11px] text-slate-500">{perm.description}</p>
                        </div>
                      ))}
                    </div>
                  </div>
                ))}

                <div className="border-t border-slate-300 mt-3 pt-2">
                  <div className="flex flex-col gap-1">
                    {MEMBER_ROLES.map((r) => (
                      <button
                        key={r}
                        type="button"
                        onClick={() => {
                          setRole(r as MemberRole);
                          closePopover();
                        }}
                        className={
                          "w-full text-left px-3 py-2 rounded hover:bg-slate-100 " +
                          (role === r ? "bg-slate-100 font-semibold" : "")
                        }
                      >
                        {roleLabel(r)}
                      </button>
                    ))}
                  </div>
                </div>
              </PopoverContent>
            </Popover>

            {/* hidden select for validation */}
            <select
              aria-hidden="true"
              tabIndex={-1}
              required
              value={role}
              onChange={(e) => setRole(e.target.value as MemberRole)}
              style={{
                position: "absolute",
                opacity: 0,
                pointerEvents: "none",
                height: 0,
                width: 0,
              }}
            >
              {MEMBER_ROLES.map((r) => (
                <option key={r} value={r}>
                  {roleLabel(r)}
                </option>
              ))}
            </select>
          </div>

          {/* 権限・ブランド */}
          <div>
            <label className="block text-sm text-slate-300 mb-1">
              権限名（カンマ区切り）
            </label>
            <Input
              variant="default"
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
            <Input
              variant="default"
              className="w-full rounded border border-slate-600 bg-slate-800 px-3 py-2"
              value={brandsText}
              onChange={(e) => setBrandsText(e.target.value)}
              placeholder="LUMINA, NEXUS"
            />
          </div>
        </form>
      </div>
    </PageStyle>
  );
}
