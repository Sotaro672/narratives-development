// frontend/console/shell/src/auth/hook/useAuthPage.ts
import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAuthActions } from "../../application/useAuthActions";

export type AuthMode = "signup" | "signin";

export function useAuthPage() {
  const navigate = useNavigate();
  const { signUp, signIn, submitting, error, setError } = useAuthActions();

  // -------------------------
  // モード
  // -------------------------
  const [mode, setMode] = useState<AuthMode>("signin");

  // -------------------------
  // 入力値
  // -------------------------
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");

  const [lastName, setLastName] = useState("");
  const [firstName, setFirstName] = useState("");
  const [lastNameKana, setLastNameKana] = useState("");
  const [firstNameKana, setFirstNameKana] = useState("");

  const [companyName, setCompanyName] = useState("");

  // -------------------------
  // 新規登録フロー管理
  // -------------------------
  const [signupRequested, setSignupRequested] = useState(false);
  const [signupCompleted, setSignupCompleted] = useState(false);

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
      setSignupRequested(false);
      setSignupCompleted(false);
    },
    [resetForm],
  );

  // -------------------------
  // submit handler
  // -------------------------
  const handleFormSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();

      if (mode === "signup") {
        if (password !== confirmPassword) {
          setError("パスワードが一致していません。");
          return;
        }

        setSignupRequested(true);
        setSignupCompleted(false);

        await signUp(email, password, {
          lastName,
          firstName,
          lastNameKana,
          firstNameKana,
          companyName,
        });
        return;
      }

      // login
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

  // -------------------------
  // signup 完了判定
  // signupRequested=true && submitting=false && error=null
  // 完了後 → CertificationPage へ遷移
  // -------------------------
  useEffect(() => {
    if (mode !== "signup") return;

    if (signupRequested && !submitting && !error) {
      setSignupCompleted(true);

      // フローをリセットして同ページ戻り時の無限リダイレクトを防止
      setSignupRequested(false);
    }
  }, [mode, signupRequested, submitting, error, navigate]);

  const resetSignupFlow = useCallback(() => {
    setSignupRequested(false);
    setSignupCompleted(false);
  }, []);

  return {
    // モード
    mode,
    switchMode,

    // 入力
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

    // 状態
    submitting,
    error,
    setError,

    // サインアップフロー
    signupRequested,
    signupCompleted,
    resetSignupFlow,

    // submit ラッパ
    handleFormSubmit,
  };
}
