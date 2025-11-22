// frontend/console/shell/src/auth/presentation/pages/InvitationPage.tsx
import * as React from "react";
import { useSearchParams } from "react-router-dom";
import PageStyle from "../../../layout/PageStyle/PageStyle";
import { Input } from "../../../shared/ui/input";
import { useInvitationPage } from "../hook/useInvitationPage";

/**
 * 招待ページ（氏名・かな + パスワード + 割り当て情報）
 */
export default function InvitationPage() {
  // ★ URL クエリ (?token=INV_xxx) を取得
  const [searchParams] = useSearchParams();
  const invitationToken = searchParams.get("token") ?? "";

  const {
    formRef,

    // 氏名情報
    lastName,
    setLastName,
    lastNameKana,
    setLastNameKana,
    firstName,
    setFirstName,
    firstNameKana,
    setFirstNameKana,

    // パスワード
    password,
    setPassword,
    passwordConfirm,
    setPasswordConfirm,

    // 割り当て情報（メールと同じ内容を表示用に保持）
    companyId,
    assignedBrandIds,
    permissions,

    // token setter（useInvitationPage 内に追加してある前提）
    setToken,

    // Actions
    handleBack,
    handleCreate,
    handleSubmit,
  } = useInvitationPage();

  // ★ 最初の一度だけ token をセット
  React.useEffect(() => {
    if (invitationToken) {
      setToken(invitationToken);
    }
  }, [invitationToken, setToken]);

  return (
    <PageStyle title="メンバー招待" onBack={handleBack} onCreate={handleCreate}>
      <div className="p-4 max-w-3xl mx-auto">
        <form ref={formRef} onSubmit={handleSubmit} className="space-y-4" noValidate>
          {/* 姓 → 姓（かな） */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm text-slate-300 mb-1">姓</label>
              <Input
                variant="default"
                className="w-full rounded border border-slate-600 bg-slate-800 px-3 py-2"
                value={lastName}
                onChange={(e) => setLastName(e.target.value)}
                placeholder="山田"
              />
            </div>
            <div>
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
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm text-slate-300 mb-1">名</label>
              <Input
                variant="default"
                className="w-full rounded border border-slate-600 bg-slate-800 px-3 py-2"
                value={firstName}
                onChange={(e) => setFirstName(e.target.value)}
                placeholder="太郎"
              />
            </div>
            <div>
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

          {/* パスワード */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm text-slate-300 mb-1">パスワード</label>
              <Input
                type="password"
                variant="default"
                className="w-full rounded border border-slate-600 bg-slate-800 px-3 py-2"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="8文字以上"
              />
            </div>

            <div>
              <label className="block text-sm text-slate-300 mb-1">
                パスワード（確認用）
              </label>
              <Input
                type="password"
                variant="default"
                className="w-full rounded border border-slate-600 bg-slate-800 px-3 py-2"
                value={passwordConfirm}
                onChange={(e) => setPasswordConfirm(e.target.value)}
                placeholder="もう一度入力"
              />
            </div>
          </div>

          {/* 割り当て情報（表示専用） */}
          <div className="mt-4 space-y-3">
            <h2 className="text-sm font-semibold text-slate-200">割り当て情報</h2>

            <div>
              <label className="block text-sm text-slate-300 mb-1">Company ID</label>
              <p className="text-sm text-slate-100 bg-slate-900 rounded px-3 py-2 border border-slate-700">
                {companyId || "-"}
              </p>
            </div>

            <div>
              <label className="block text-sm text-slate-300 mb-1">Assigned Brands</label>
              <p className="text-sm text-slate-100 bg-slate-900 rounded px-3 py-2 border border-slate-700 whitespace-pre-wrap break-all">
                {assignedBrandIds || "-"}
              </p>
            </div>

            <div>
              <label className="block text-sm text-slate-300 mb-1">Permissions</label>
              <p className="text-sm text-slate-100 bg-slate-900 rounded px-3 py-2 border border-slate-700 whitespace-pre-wrap break-all">
                {permissions || "-"}
              </p>
            </div>
          </div>
        </form>
      </div>
    </PageStyle>
  );
}
