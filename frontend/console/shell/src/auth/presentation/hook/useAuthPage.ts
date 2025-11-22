//frontend\console\shell\src\auth\presentation\hook\useAuthPage.ts
import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAuthActions } from "../../application/useAuthActions";
import { sendPasswordResetEmail } from "firebase/auth";
import { auth } from "../../infrastructure/config/firebaseClient";

export type AuthMode = "signup" | "signin";

// -------------------------
// かな関連ヘルパ
// -------------------------

// ひらがな・カタカナ・半角カナをひらがなに寄せる（削除はしない）
function toHiragana(input: string): string {
  if (!input) return "";

  let s = input;

  // 全角カタカナ → ひらがな
  s = s.replace(/[\u30A1-\u30F6]/g, (ch) =>
    String.fromCharCode(ch.charCodeAt(0) - 0x60),
  );

  // 半角カナ → 全角カナ → ひらがな（簡易変換）
  s = s.replace(/[\uff61-\uff9f]/g, (ch) => {
    const kataCode = ch.charCodeAt(0) - 0xff61 + 0x30a1;
    const hiraCode = kataCode - 0x60;
    return String.fromCharCode(hiraCode);
  });

  return s;
}

// 「ひらがな + スペースのみか」をチェック
function isHiraganaOnly(input: string): boolean {
  if (!input) return false;
  return /^[\u3041-\u3096\s]+$/.test(input);
}

// -------------------------
// 会社名の正規化
//   ※ アルファベットも許可するので、ここでは不要な削除は行わない
// -------------------------
function normalizeCompanyName(input: string): string {
  if (!input) return "";
  // 必要ならここで trim や 連続スペースの正規化などだけを行う
  return input;
}

export function useAuthPage() {
  const navigate = useNavigate();
  const { signUp, signIn, submitting, error, setError } = useAuthActions();

  // -------------------------
  // モード
  // -------------------------
  const [mode, setMode] = useState<AuthMode>("signin");

  // -------------------------
  // 「パスワードをお忘れの方」モード
  // -------------------------
  const [forgotPasswordMode, setForgotPasswordMode] = useState(false);

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

  const [companyName, _setCompanyName] = useState("");

  // 会社名（アルファベットも含めそのまま保持・必要なら軽い正規化のみ）
  const setCompanyName = (v: string) => _setCompanyName(normalizeCompanyName(v));

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

    _setCompanyName("");

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
    async (e: React.FormEvent) => {
      e.preventDefault();

      // ▼ パスワードをお忘れの方（signin + forgotPasswordMode）
      if (mode === "signin" && forgotPasswordMode) {
        if (!email.trim()) {
          setError("パスワード再設定メールを送るメールアドレスを入力してください。");
          return;
        }

        try {
          await sendPasswordResetEmail(auth, email.trim());
          window.alert(
            "パスワード再設定用のメールを送信しました。\nメールに記載されたリンクからパスワードを再設定してください。",
          );
          setForgotPasswordMode(false);
          setError(null);
        } catch (err: any) {
          console.error("[useAuthPage] sendPasswordResetEmail error:", err);
          setError(
            "パスワード再設定メールの送信に失敗しました。メールアドレスをご確認ください。",
          );
        }
        return;
      }

      if (mode === "signup") {
        if (password !== confirmPassword) {
          setError("パスワードが一致していません。");
          return;
        }

        // かな入力チェック（カタカナ/半角カナもひらがなへ正規化してから判定）
        const normalizedLastKana = toHiragana(lastNameKana.trim());
        const normalizedFirstKana = toHiragana(firstNameKana.trim());

        if (
          !isHiraganaOnly(normalizedLastKana) ||
          !isHiraganaOnly(normalizedFirstKana)
        ) {
          setError("姓・名のかなはひらがなのみで入力してください。");
          return;
        }

        setSignupRequested(true);
        setSignupCompleted(false);

        await signUp(email, password, {
          lastName,
          firstName,
          lastNameKana: normalizedLastKana,
          firstNameKana: normalizedFirstKana,
          companyName,
        });
        return;
      }

      // ▼ 通常ログイン
      await signIn(email, password);
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
  // signup 完了判定
  // -------------------------
  useEffect(() => {
    if (mode !== "signup") return;

    if (signupRequested && !submitting && !error) {
      setSignupCompleted(true);
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
    signupRequested,
    signupCompleted,
    resetSignupFlow,

    // submit ラッパ
    handleFormSubmit,
  };
}
