// frontend/console/shell/src/auth/presentation/hook/useAuthPage.ts

import {
  useCallback,
  useEffect,
  useState,
} from "react";
import {
  sendPasswordResetEmail,
} from "firebase/auth";

import {
  useAuthActions,
} from "../../application/useAuthActions";
import {
  auth,
} from "../../infrastructure/config/firebaseClient";

export type AuthMode =
  | "signup"
  | "signin";

// -------------------------
// かな関連ヘルパ
// -------------------------

// 「ひらがな + スペースのみか」をチェック
function isHiraganaOnly(
  input: string,
): boolean {
  if (!input) {
    return false;
  }

  return /^[\u3041-\u3096\s]+$/.test(
    input,
  );
}

export function useAuthPage() {
  const {
    signUp,
    signIn,
    submitting,
    error,
    setError,
  } = useAuthActions();

  // -------------------------
  // モード
  // -------------------------
  const [mode, setMode] =
    useState<AuthMode>("signin");

  // -------------------------
  // 「パスワードをお忘れの方」モード
  // -------------------------
  const [
    forgotPasswordMode,
    setForgotPasswordMode,
  ] = useState(false);

  // -------------------------
  // 入力値
  // -------------------------
  const [email, setEmail] =
    useState("");

  const [password, setPassword] =
    useState("");

  const [
    confirmPassword,
    setConfirmPassword,
  ] = useState("");

  const [lastName, setLastName] =
    useState("");

  const [firstName, setFirstName] =
    useState("");

  const [
    lastNameKana,
    setLastNameKana,
  ] = useState("");

  const [
    firstNameKana,
    setFirstNameKana,
  ] = useState("");

  const [
    companyName,
    setCompanyName,
  ] = useState("");

  // -------------------------
  // 新規登録フロー管理
  // -------------------------
  const [
    signupRequested,
    setSignupRequested,
  ] = useState(false);

  const [
    signupCompleted,
    setSignupCompleted,
  ] = useState(false);

  const resetForm = useCallback(() => {
    setEmail("");
    setPassword("");
    setConfirmPassword("");

    setLastName("");
    setFirstName("");
    setLastNameKana("");
    setFirstNameKana("");

    setCompanyName("");

    setForgotPasswordMode(false);
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
    async (
      event: React.FormEvent,
    ) => {
      event.preventDefault();

      // パスワードをお忘れの方
      if (
        mode === "signin" &&
        forgotPasswordMode
      ) {
        const normalizedEmail =
          email.trim();

        if (!normalizedEmail) {
          setError(
            "パスワード再設定メールを送るメールアドレスを入力してください。",
          );
          return;
        }

        try {
          await sendPasswordResetEmail(
            auth,
            normalizedEmail,
          );

          window.alert(
            "パスワード再設定用のメールを送信しました。\nメールに記載されたリンクからパスワードを再設定してください。",
          );

          setForgotPasswordMode(false);
          setError(null);
        } catch (error: unknown) {
          console.error(
            "[useAuthPage] sendPasswordResetEmail error:",
            error,
          );

          setError(
            "パスワード再設定メールの送信に失敗しました。メールアドレスをご確認ください。",
          );
        }

        return;
      }

      // 新規登録
      if (mode === "signup") {
        if (
          password !==
          confirmPassword
        ) {
          setError(
            "パスワードが一致していません。",
          );
          return;
        }

        const lastKana =
          lastNameKana.trim();

        const firstKana =
          firstNameKana.trim();

        if (
          !isHiraganaOnly(lastKana) ||
          !isHiraganaOnly(firstKana)
        ) {
          setError(
            "姓・名のかなはひらがなのみで入力してください。",
          );
          return;
        }

        setSignupRequested(true);
        setSignupCompleted(false);

        await signUp(
          email,
          password,
          {
            lastName,
            firstName,
            lastNameKana: lastKana,
            firstNameKana: firstKana,
            companyName,
          },
        );

        return;
      }

      // 通常ログイン
      await signIn(
        email,
        password,
      );
    },
    [
      mode,
      forgotPasswordMode,
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
  // signup完了判定
  // -------------------------
  useEffect(() => {
    if (mode !== "signup") {
      return;
    }

    if (
      signupRequested &&
      !submitting &&
      !error
    ) {
      setSignupCompleted(true);
      setSignupRequested(false);
    }
  }, [
    mode,
    signupRequested,
    submitting,
    error,
  ]);

  const resetSignupFlow =
    useCallback(() => {
      setSignupRequested(false);
      setSignupCompleted(false);
    }, []);

  return {
    // モード
    mode,
    switchMode,

    // 「パスワードをお忘れの方」モード
    forgotPasswordMode,
    setForgotPasswordMode,

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
    signupCompleted,
    resetSignupFlow,

    // submit
    handleFormSubmit,
  };
}