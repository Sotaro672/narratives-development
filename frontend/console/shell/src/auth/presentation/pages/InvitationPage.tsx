// frontend/console/shell/src/auth/presentation/pages/InvitationPage.tsx
// frontend/console/shell/src/auth/presentation/pages/InvitationPage.tsx
import * as React from "react";
import { useSearchParams } from "react-router-dom";
import { Input } from "../../../shared/ui/input";
import { useInvitationPage } from "../hook/useInvitationPage";

/**
 * 招待ページ（画面幅いっぱい・白背景）
 * - 上部に説明文
 * - 割り当て情報 → 氏名 → パスワード
 * - ボタンは画面下部中央
 */
export default function InvitationPage() {
  const [searchParams] = useSearchParams();
  const invitationToken = searchParams.get("token") ?? "";

  const {
    formRef,
    email,
    lastName,
    setLastName,
    lastNameKana,
    setLastNameKana,
    firstName,
    setFirstName,
    firstNameKana,
    setFirstNameKana,
    password,
    setPassword,
    passwordConfirm,
    setPasswordConfirm,
    companyId,
    assignedBrandIds,
    permissions,
    companyName,
    assignedBrandNames,
    setToken,
    handleSubmit,
  } = useInvitationPage();

  React.useEffect(() => {
    if (invitationToken) {
      setToken(invitationToken);
    }
  }, [invitationToken, setToken]);

  const companyText = companyName || companyId || "-";

  const assignedBrandText =
    assignedBrandNames.length > 0
      ? assignedBrandNames.join(", ")
      : assignedBrandIds.length > 0
        ? assignedBrandIds.join(", ")
        : "-";

  const permissionsText =
    permissions.length > 0 ? permissions.join(", ") : "-";

  const emailText = email || "-";

  React.useEffect(() => {
    // eslint-disable-next-line no-console
    console.log("[InvitationPage] invitation info", {
      token: invitationToken,
      companyId,
      companyName,
      assignedBrandIds,
      assignedBrandNames,
      permissions,
      email,
    });
  }, [
    invitationToken,
    companyId,
    companyName,
    assignedBrandIds,
    assignedBrandNames,
    permissions,
    email,
  ]);

  return (
    <div className="min-h-screen bg-white text-slate-900 flex flex-col">
      <main className="flex-1 max-w-3xl w-full mx-auto px-4 py-10 flex flex-col">
        <p className="text-sm text-slate-700 mb-6">
          招待内容を確認し、氏名とパスワードを設定してください。
        </p>

        <form
          ref={formRef}
          onSubmit={handleSubmit}
          className="space-y-6 flex-1 flex flex-col"
          noValidate
        >
          <section className="space-y-3">

            <div>
              <label className="block text-sm text-slate-600 mb-1">
                会社名
              </label>
              <p className="text-sm text-slate-900 bg-white rounded px-3 py-2 border border-slate-300">
                {companyText}
              </p>
            </div>

            <div>
              <label className="block text-sm text-slate-600 mb-1">
                割り当てブランド
              </label>
              <p className="text-sm text-slate-900 bg-white rounded px-3 py-2 border border-slate-300 whitespace-pre-wrap break-all">
                {assignedBrandText}
              </p>
            </div>

            <div>
              <label className="block text-sm text-slate-600 mb-1">
                権限
              </label>
              <p className="text-sm text-slate-900 bg-white rounded px-3 py-2 border border-slate-300 whitespace-pre-wrap break-all">
                {permissionsText}
              </p>
            </div>

            <div>
              <label className="block text-sm text-slate-600 mb-1">
                メールアドレス
              </label>
              <p className="text-sm text-slate-900 bg-white rounded px-3 py-2 border border-slate-300 break-all">
                {emailText}
              </p>
            </div>
          </section>

          <section className="space-y-4">
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-sm text-slate-600 mb-1">姓</label>
                <Input
                  variant="default"
                  className="w-full rounded border border-slate-300 bg-white px-3 py-2"
                  value={lastName}
                  onChange={(e) => setLastName(e.target.value)}
                  placeholder="山田"
                />
              </div>
              <div>
                <label className="block text-sm text-slate-600 mb-1">
                  姓（かな）
                </label>
                <Input
                  variant="default"
                  className="w-full rounded border border-slate-300 bg-white px-3 py-2"
                  value={lastNameKana}
                  onChange={(e) => setLastNameKana(e.target.value)}
                  placeholder="やまだ"
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-sm text-slate-600 mb-1">名</label>
                <Input
                  variant="default"
                  className="w-full rounded border border-slate-300 bg-white px-3 py-2"
                  value={firstName}
                  onChange={(e) => setFirstName(e.target.value)}
                  placeholder="太郎"
                />
              </div>
              <div>
                <label className="block text-sm text-slate-600 mb-1">
                  名（かな）
                </label>
                <Input
                  variant="default"
                  className="w-full rounded border border-slate-300 bg-white px-3 py-2"
                  value={firstNameKana}
                  onChange={(e) => setFirstNameKana(e.target.value)}
                  placeholder="たろう"
                />
              </div>
            </div>
          </section>

          <section className="space-y-4">
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-sm text-slate-600 mb-1">
                  パスワード
                </label>
                <Input
                  type="password"
                  variant="default"
                  className="w-full rounded border border-slate-300 bg-white px-3 py-2"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="8文字以上"
                />
              </div>

              <div>
                <label className="block text-sm text-slate-600 mb-1">
                  パスワード（確認用）
                </label>
                <Input
                  type="password"
                  variant="default"
                  className="w-full rounded border border-slate-300 bg-white px-3 py-2"
                  value={passwordConfirm}
                  onChange={(e) => setPasswordConfirm(e.target.value)}
                  placeholder="もう一度入力"
                />
              </div>
            </div>
          </section>

          <div className="flex-1" />

          <div className="flex justify-center mt-6 mb-8">
            <button
              type="submit"
              className="px-10 py-2 rounded-full bg-slate-900 text-white text-sm font-semibold shadow hover:bg-black transition"
            >
              サインイン
            </button>
          </div>
        </form>
      </main>
    </div>
  );
}