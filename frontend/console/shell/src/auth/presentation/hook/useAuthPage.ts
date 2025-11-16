// frontend/console/shell/src/auth/hook/useAuthPage.ts
import { useCallback, useState } from "react";
import { useAuthActions } from "../../application/useAuthActions";

export type AuthMode = "signup" | "signin";

export function useAuthPage() {
  const { signUp, signIn, submitting, error, setError } = useAuthActions();

  const [mode, setMode] = useState<AuthMode>("signin");

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");

  // 姓名＋かな
  const [lastName, setLastName] = useState("");
  const [firstName, setFirstName] = useState("");
  const [lastNameKana, setLastNameKana] = useState("");
  const [firstNameKana, setFirstNameKana] = useState("");

  // 会社名・団体名（signup 時のみ使用 / 任意入力）
  const [companyName, setCompanyName] = useState("");

  const resetForm = useCallback(() => {
    setEmail("");
    setPassword("");
    setConfirmPassword("");
    setLastName("");
    setFirstName("");
    setLastNameKana("");
    setFirstNameKana("");
    setCompanyName("");
    setError(null);
  }, [setError]);

  const switchMode = useCallback(
    (next: AuthMode) => {
      setMode(next);
      resetForm();
    },
    [resetForm],
  );

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();

      if (mode === "signup") {
        if (password !== confirmPassword) {
          setError("パスワードが一致していません。");
          return;
        }

        await signUp(email, password, {
          lastName,
          firstName,
          lastNameKana,
          firstNameKana,
          companyName,
        });
        return;
      }

      // ログイン(Sign In)
      await signIn(email, password);
    },
    [
      mode,
      email,
      password,
      confirmPassword,
      lastName,
      firstName,
      lastNameKana,
      firstNameKana,
      companyName,
      signUp,
      signIn,
      setError,
    ],
  );

  return {
    // モード
    mode,
    switchMode,

    // 入力値と setter
    email,
    setEmail,
    password,
    setPassword,
    confirmPassword,
    setConfirmPassword,
    lastName,
    setLastName,
    firstName,
    setFirstName,
    lastNameKana,
    setLastNameKana,
    firstNameKana,
    setFirstNameKana,
    companyName,
    setCompanyName,

    // 認証アクションの状態
    submitting,
    error,
    setError,

    // submit ハンドラ
    handleSubmit,
  };
}
