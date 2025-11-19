// frontend/console/shell/src/auth/presentation/hook/useAuthPage.ts
import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAuthActions } from "../../application/useAuthActions";
import { sendPasswordResetEmail } from "firebase/auth";
import { auth } from "../../infrastructure/config/firebaseClient";

export type AuthMode = "signup" | "signin";

// -------------------------
// カナ関連ヘルパ
// -------------------------

// ひらがな・半角カナを全角カタカナに寄せる（削除はしない）
function toKatakana(input: string): string {
  if (!input) return "";

  let s = input;

  // ひらがな → カタカナ
  s = s.replace(/[\u3041-\u3096]/g, (ch) =>
    String.fromCharCode(ch.charCodeAt(0) + 0x60),
  );

  // 半角カナ → 全角カナ（簡易変換）
  s = s.replace(/[\uff61-\uff9f]/g, (ch) => {
    const code = ch.charCodeAt(0) - 0xff61 + 0x30a1;
    return String.fromCharCode(code);
  });

  return s;
}

// 「全角カタカナ + 長音 + スペースのみか」をチェック
function isKatakanaOnly(input: string): boolean {
  if (!input) return false;
  return /^[\u30A0-\u30FFー\s]+$/.test(input);
}

// -------------------------
// 会社名からアルファベット除去
// -------------------------
function normalizeCompanyName(input: string): string {
  if (!input) return "";

  // 許可: 漢字・ひらがな・カタカナ・数字・スペース・長音
  // アルファベットだけ除去
  return input.replace(/[A-Za-z]/g, "");
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

  // 会社名アルファベット排除
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

        // カナ入力チェック（ひらがな/半角カナはカタカナ化してから判定）
        const normalizedLastKana = toKatakana(lastNameKana.trim());
        const normalizedFirstKana = toKatakana(firstNameKana.trim());

        if (
          !isKatakanaOnly(normalizedLastKana) ||
          !isKatakanaOnly(normalizedFirstKana)
        ) {
          setError("姓・名のカナは全角カタカナのみで入力してください。");
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
