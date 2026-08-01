// frontend/console/shell/src/pages/InvitationPage.tsx

import * as React from "react";
import {
  useSearchParams,
} from "react-router-dom";

import {
  useInvitationPage,
} from "../auth/presentation/hook/useInvitationPage";
import {
  Input,
} from "../shared/ui/input";

/**
 * 招待ページ
 * - 招待情報を表示
 * - メールアドレス、氏名、パスワードを設定
 * - 招待完了後にトップページへ遷移
 */
export default function InvitationPage() {
  const [
    searchParams,
  ] = useSearchParams();

  const invitationToken =
    searchParams.get("token") ?? "";

  const {
    // 招待トークン
    setToken,

    // 処理状態
    loading,
    loadingInvitationInfo,
    submitting,
    error,

    // メールアドレス
    email,
    setEmail,

    // 氏名
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

    // 招待情報
    companyName,
    assignedBrandNames,

    // 送信
    handleSubmit,
  } = useInvitationPage();

  // -------------------------
  // URLのtokenをhookへ反映
  // -------------------------

  React.useEffect(() => {
    setToken(invitationToken);
  }, [
    invitationToken,
    setToken,
  ]);

  const companyText =
    loadingInvitationInfo
      ? "読み込み中..."
      : companyName || "-";

  const assignedBrandText =
    loadingInvitationInfo
      ? "読み込み中..."
      : assignedBrandNames.length > 0
        ? assignedBrandNames.join(", ")
        : "-";

  return (
    <div className="min-h-screen bg-white text-slate-900 flex flex-col">
      <main className="flex-1 max-w-3xl w-full mx-auto px-4 py-10 flex flex-col">
        <p className="text-sm text-slate-700 mb-6">
          招待内容を確認し、メールアドレス、氏名、パスワードを設定してください。
        </p>

        {error && (
          <div
            role="alert"
            className="mb-6 rounded border border-red-300 bg-red-50 px-4 py-3 text-sm text-red-700"
          >
            {error}
          </div>
        )}

        <form
          onSubmit={handleSubmit}
          className="space-y-6 flex-1 flex flex-col"
          aria-busy={loading}
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
          </section>

          <section className="space-y-4">
            <div>
              <label className="block text-sm text-slate-600 mb-1">
                メールアドレス
              </label>

              <Input
                type="email"
                autoComplete="email"
                variant="default"
                className="w-full rounded border border-slate-300 bg-white px-3 py-2"
                value={email}
                onChange={(event) =>
                  setEmail(
                    event.target.value,
                  )
                }
                placeholder="example@example.com"
                disabled={submitting}
              />
            </div>
          </section>

          <section className="space-y-4">
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-sm text-slate-600 mb-1">
                  姓
                </label>

                <Input
                  autoComplete="family-name"
                  variant="default"
                  className="w-full rounded border border-slate-300 bg-white px-3 py-2"
                  value={lastName}
                  onChange={(event) =>
                    setLastName(
                      event.target.value,
                    )
                  }
                  placeholder="山田"
                  disabled={submitting}
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
                  onChange={(event) =>
                    setLastNameKana(
                      event.target.value,
                    )
                  }
                  placeholder="やまだ"
                  disabled={submitting}
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-sm text-slate-600 mb-1">
                  名
                </label>

                <Input
                  autoComplete="given-name"
                  variant="default"
                  className="w-full rounded border border-slate-300 bg-white px-3 py-2"
                  value={firstName}
                  onChange={(event) =>
                    setFirstName(
                      event.target.value,
                    )
                  }
                  placeholder="太郎"
                  disabled={submitting}
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
                  onChange={(event) =>
                    setFirstNameKana(
                      event.target.value,
                    )
                  }
                  placeholder="たろう"
                  disabled={submitting}
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
                  autoComplete="new-password"
                  variant="default"
                  className="w-full rounded border border-slate-300 bg-white px-3 py-2"
                  value={password}
                  onChange={(event) =>
                    setPassword(
                      event.target.value,
                    )
                  }
                  placeholder="8文字以上"
                  disabled={submitting}
                />
              </div>

              <div>
                <label className="block text-sm text-slate-600 mb-1">
                  パスワード（確認用）
                </label>

                <Input
                  type="password"
                  autoComplete="new-password"
                  variant="default"
                  className="w-full rounded border border-slate-300 bg-white px-3 py-2"
                  value={passwordConfirm}
                  onChange={(event) =>
                    setPasswordConfirm(
                      event.target.value,
                    )
                  }
                  placeholder="もう一度入力"
                  disabled={submitting}
                />
              </div>
            </div>
          </section>

          <div className="flex-1" />

          <div className="flex justify-center mt-6 mb-8">
            <button
              type="submit"
              className="px-10 py-2 rounded-full bg-slate-900 text-white text-sm font-semibold shadow transition hover:bg-black disabled:cursor-not-allowed disabled:opacity-60"
              disabled={loading}
            >
              {submitting
                ? "処理中..."
                : loadingInvitationInfo
                  ? "招待情報を確認中..."
                  : "サインイン"}
            </button>
          </div>
        </form>
      </main>
    </div>
  );
}