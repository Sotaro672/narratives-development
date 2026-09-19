// frontend/console/shell/src/pages/InvitationPage.tsx

import * as React from "react";
import { useSearchParams } from "react-router-dom";

import { useInvitationPage } from "../auth/presentation/hook/useInvitationPage";
import { ErrorMessage } from "../shared/ui/error";
import { Input } from "../shared/ui/input";
import { Label } from "../shared/ui/label";

import "../styles/member.css";

/**
 * 招待ページ
 * - 招待情報を表示
 * - メールアドレス、氏名、パスワードを設定
 * - 招待完了後にトップページへ遷移
 */
export default function InvitationPage() {
  const [searchParams] = useSearchParams();

  const invitationToken = searchParams.get("token") ?? "";

  const {
    setToken,
    loading,
    loadingInvitationInfo,
    submitting,
    error,
    email,
    setEmail,
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
    companyName,
    assignedBrandNames,
    handleSubmit,
  } = useInvitationPage();

  React.useEffect(() => {
    setToken(invitationToken);
  }, [invitationToken, setToken]);

  const companyText = loadingInvitationInfo
    ? "読み込み中..."
    : companyName || "-";

  const assignedBrandText = loadingInvitationInfo
    ? "読み込み中..."
    : assignedBrandNames.length > 0
      ? assignedBrandNames.join(", ")
      : "-";

  return (
    <div className="invitation-page">
      <main className="invitation-page__main">
        <p className="invitation-page__description">
          招待内容を確認し、メールアドレス、氏名、パスワードを設定してください。
        </p>

        {error && (
          <ErrorMessage
            variant="panel"
            className="invitation-page__error"
          >
            {error}
          </ErrorMessage>
        )}

        <form
          onSubmit={handleSubmit}
          className="invitation-page__form"
          aria-busy={loading}
          noValidate
        >
          <section className="invitation-page__section invitation-page__section--compact">
            <div>
              <Label className="invitation-page__label">
                会社名
              </Label>

              <p className="invitation-page__readonly-value">
                {companyText}
              </p>
            </div>

            <div>
              <Label className="invitation-page__label">
                割り当てブランド
              </Label>

              <p className="invitation-page__readonly-value invitation-page__readonly-value--brands">
                {assignedBrandText}
              </p>
            </div>
          </section>

          <section className="invitation-page__section">
            <div>
              <Label
                htmlFor="invitation-email"
                className="invitation-page__label"
              >
                メールアドレス
              </Label>

              <Input
                id="invitation-email"
                type="email"
                autoComplete="email"
                variant="default"
                className="invitation-page__input"
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                placeholder="example@example.com"
                disabled={submitting}
              />
            </div>
          </section>

          <section className="invitation-page__section">
            <div className="invitation-page__grid">
              <div>
                <Label
                  htmlFor="invitation-last-name"
                  className="invitation-page__label"
                >
                  姓
                </Label>

                <Input
                  id="invitation-last-name"
                  autoComplete="family-name"
                  variant="default"
                  className="invitation-page__input"
                  value={lastName}
                  onChange={(event) => setLastName(event.target.value)}
                  placeholder="山田"
                  disabled={submitting}
                />
              </div>

              <div>
                <Label
                  htmlFor="invitation-last-name-kana"
                  className="invitation-page__label"
                >
                  姓（かな）
                </Label>

                <Input
                  id="invitation-last-name-kana"
                  variant="default"
                  className="invitation-page__input"
                  value={lastNameKana}
                  onChange={(event) => setLastNameKana(event.target.value)}
                  placeholder="やまだ"
                  disabled={submitting}
                />
              </div>
            </div>

            <div className="invitation-page__grid">
              <div>
                <Label
                  htmlFor="invitation-first-name"
                  className="invitation-page__label"
                >
                  名
                </Label>

                <Input
                  id="invitation-first-name"
                  autoComplete="given-name"
                  variant="default"
                  className="invitation-page__input"
                  value={firstName}
                  onChange={(event) => setFirstName(event.target.value)}
                  placeholder="太郎"
                  disabled={submitting}
                />
              </div>

              <div>
                <Label
                  htmlFor="invitation-first-name-kana"
                  className="invitation-page__label"
                >
                  名（かな）
                </Label>

                <Input
                  id="invitation-first-name-kana"
                  variant="default"
                  className="invitation-page__input"
                  value={firstNameKana}
                  onChange={(event) => setFirstNameKana(event.target.value)}
                  placeholder="たろう"
                  disabled={submitting}
                />
              </div>
            </div>
          </section>

          <section className="invitation-page__section">
            <div className="invitation-page__grid">
              <div>
                <Label
                  htmlFor="invitation-password"
                  className="invitation-page__label"
                >
                  パスワード
                </Label>

                <Input
                  id="invitation-password"
                  type="password"
                  autoComplete="new-password"
                  variant="default"
                  className="invitation-page__input"
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  placeholder="8文字以上"
                  disabled={submitting}
                />
              </div>

              <div>
                <Label
                  htmlFor="invitation-password-confirm"
                  className="invitation-page__label"
                >
                  パスワード（確認用）
                </Label>

                <Input
                  id="invitation-password-confirm"
                  type="password"
                  autoComplete="new-password"
                  variant="default"
                  className="invitation-page__input"
                  value={passwordConfirm}
                  onChange={(event) => setPasswordConfirm(event.target.value)}
                  placeholder="もう一度入力"
                  disabled={submitting}
                />
              </div>
            </div>
          </section>

          <div className="invitation-page__spacer" />

          <div className="invitation-page__actions">
            <button
              type="submit"
              className="invitation-page__submit"
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