// frontend/console/shell/src/pages/InvitationPage.tsx

import * as React from "react";
import { useSearchParams } from "react-router-dom";

import { useInvitationPage } from "../auth/presentation/hook/useInvitationPage";
import { Button } from "../shared/ui/button";
import {
  CardField,
  CardFields,
  CardReadonly,
} from "../shared/ui/card";
import { ErrorMessage } from "../shared/ui/error";
import { Input } from "../shared/ui/input";
import { Label } from "../shared/ui/label";
import Stack from "../shared/ui/stack";
import Text from "../shared/ui/text";

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
        <Text
          as="p"
          size="sm"
          className="invitation-page__description"
        >
          招待内容を確認し、メールアドレス、氏名、パスワードを設定してください。
        </Text>

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
          <Stack gap="lg">
            <section>
              <Stack gap="md">
                <CardField>
                  <Label>会社名</Label>
                  <CardReadonly>
                    {companyText}
                  </CardReadonly>
                </CardField>

                <CardField>
                  <Label>割り当てブランド</Label>
                  <CardReadonly>
                    {assignedBrandText}
                  </CardReadonly>
                </CardField>
              </Stack>
            </section>

            <section>
              <CardField>
                <Label htmlFor="invitation-email">
                  メールアドレス
                </Label>

                <Input
                  id="invitation-email"
                  type="email"
                  autoComplete="email"
                  value={email}
                  onChange={(event) => setEmail(event.target.value)}
                  placeholder="example@example.com"
                  disabled={submitting}
                />
              </CardField>
            </section>

            <section>
              <Stack gap="md">
                <CardFields>
                  <CardField>
                    <Label htmlFor="invitation-last-name">
                      姓
                    </Label>

                    <Input
                      id="invitation-last-name"
                      autoComplete="family-name"
                      value={lastName}
                      onChange={(event) => setLastName(event.target.value)}
                      placeholder="山田"
                      disabled={submitting}
                    />
                  </CardField>

                  <CardField>
                    <Label htmlFor="invitation-last-name-kana">
                      姓（かな）
                    </Label>

                    <Input
                      id="invitation-last-name-kana"
                      value={lastNameKana}
                      onChange={(event) => setLastNameKana(event.target.value)}
                      placeholder="やまだ"
                      disabled={submitting}
                    />
                  </CardField>
                </CardFields>

                <CardFields>
                  <CardField>
                    <Label htmlFor="invitation-first-name">
                      名
                    </Label>

                    <Input
                      id="invitation-first-name"
                      autoComplete="given-name"
                      value={firstName}
                      onChange={(event) => setFirstName(event.target.value)}
                      placeholder="太郎"
                      disabled={submitting}
                    />
                  </CardField>

                  <CardField>
                    <Label htmlFor="invitation-first-name-kana">
                      名（かな）
                    </Label>

                    <Input
                      id="invitation-first-name-kana"
                      value={firstNameKana}
                      onChange={(event) => setFirstNameKana(event.target.value)}
                      placeholder="たろう"
                      disabled={submitting}
                    />
                  </CardField>
                </CardFields>
              </Stack>
            </section>

            <section>
              <CardFields>
                <CardField>
                  <Label htmlFor="invitation-password">
                    パスワード
                  </Label>

                  <Input
                    id="invitation-password"
                    type="password"
                    autoComplete="new-password"
                    value={password}
                    onChange={(event) => setPassword(event.target.value)}
                    placeholder="8文字以上"
                    disabled={submitting}
                  />
                </CardField>

                <CardField>
                  <Label htmlFor="invitation-password-confirm">
                    パスワード（確認用）
                  </Label>

                  <Input
                    id="invitation-password-confirm"
                    type="password"
                    autoComplete="new-password"
                    value={passwordConfirm}
                    onChange={(event) => setPasswordConfirm(event.target.value)}
                    placeholder="もう一度入力"
                    disabled={submitting}
                  />
                </CardField>
              </CardFields>
            </section>
          </Stack>

          <div className="invitation-page__spacer" />

          <div className="invitation-page__actions">
            <Button
              type="submit"
              variant="solid"
              size="lg"
              disabled={loading}
            >
              {submitting
                ? "処理中..."
                : loadingInvitationInfo
                  ? "招待情報を確認中..."
                  : "サインイン"}
            </Button>
          </div>
        </form>
      </main>
    </div>
  );
}