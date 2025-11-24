// frontend/console/shell/src/auth/application/useAuthActions.ts
import { useState } from "react";
import {
  createUserWithEmailAndPassword,
  signInWithEmailAndPassword,
  signOut,
} from "firebase/auth";
import { auth } from "../infrastructure/config/firebaseClient";

/**
 * 認証エラーメッセージ
 */
function messageForAuthError(code?: string): string {
  switch (code) {
    case "auth/admin-restricted-operation":
      return "現在、クライアントからの新規登録が禁止されています。Firebase Console の Authentication 設定で「ユーザー作成の許可」を有効にしてください。";
    case "auth/operation-not-allowed":
      return "Email/Password のサインイン方法が無効です。Firebase Console で有効化してください。";
    case "auth/email-already-in-use":
      return "このメールアドレスは既に登録されています。";
    case "auth/invalid-email":
      return "メールアドレスの形式が正しくありません。";
    case "auth/weak-password":
      return "パスワードが弱すぎます。より強力なパスワードを設定してください。";
    default:
      return "新規登録に失敗しました。設定を確認してください。";
  }
}

export type SignUpProfile = {
  lastName?: string;
  firstName?: string;
  lastNameKana?: string;
  firstNameKana?: string;
  companyName?: string; // 任意（会社名）
};

// ─────────────────────────────────────────────
// Backend base URL
// ─────────────────────────────────────────────
const RAW_ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined) ?? "";
const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

function sanitizeBase(u: string): string {
  return (u || "").replace(/\/+$/g, "");
}

const ENV_BASE = sanitizeBase(RAW_ENV_BASE);
const FINAL_BASE = sanitizeBase(ENV_BASE || FALLBACK_BASE);

if (!FINAL_BASE) {
  throw new Error(
    "[useAuthActions] BACKEND BASE URL is empty. Set VITE_BACKEND_BASE_URL in .env.local",
  );
}

// backend bootstrap endpoint
const BOOTSTRAP_URL = `${FINAL_BASE}/auth/bootstrap`;

// 共通 HTTP ラッパ
async function httpRequest<T>(input: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(input, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init.headers ?? {}),
    },
  });

  if (res.status === 204) return undefined as unknown as T;

  const text = await res.text().catch(() => "");

  if (!res.ok) {
    throw new Error(
      `[useAuthActions] ${res.status} ${res.statusText} :: ${text?.slice(
        0,
        300,
      )}`,
    );
  }

  try {
    return text ? (JSON.parse(text) as T) : (undefined as unknown as T);
  } catch {
    throw new Error(
      `[useAuthActions] JSON parse error. head: ${text.slice(0, 120)}`,
    );
  }
}

/**
 * ★ Bootstrap API 呼び出し
 * backend に member / company の作成を委譲する
 * （初回サインアップ専用の想定）
 */
async function callBootstrap(
  uid: string, // 実際のリクエストボディには含めない
  email: string, // 同上
  profile?: SignUpProfile,
): Promise<void> {
  const token = await auth.currentUser?.getIdToken();
  if (!token) {
    throw new Error("[useAuthActions] Not authenticated (no ID token).");
  }

  // サーバが期待する形に合わせて「プロフィールだけ」を送る
  const body = {
    lastName: (profile?.lastName ?? "").trim(),
    firstName: (profile?.firstName ?? "").trim(),
    lastNameKana: (profile?.lastNameKana ?? "").trim(),
    firstNameKana: (profile?.firstNameKana ?? "").trim(),
    companyName: (profile?.companyName ?? "").trim(),
  };

  await httpRequest<void>(BOOTSTRAP_URL, {
    method: "POST",
    body: JSON.stringify(body),
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });
}

export function useAuthActions() {
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  /**
   * サインアップ
   * - Firebase Auth でユーザー作成
   * - ログイン状態になったタイミングで backend.bootstrap を即呼び出す
   */
  async function signUp(
    email: string,
    password: string,
    profile?: SignUpProfile,
  ) {
    setSubmitting(true);
    setError(null);
    try {
      const cred = await createUserWithEmailAndPassword(auth, email, password);
      const user = cred.user;
      if (!user?.uid) {
        throw new Error("ユーザー作成後に uid を取得できませんでした。");
      }

      // 新規登録直後に bootstrap を実行
      try {
        await callBootstrap(user.uid, user.email ?? email, profile);
      } catch (e) {
        console.error("[useAuthActions] bootstrap on signUp failed:", e);
        // ここで失敗してもいったん新規登録自体は成功扱いにする
      }
    } catch (e: any) {
      console.error("signUp error", e);
      setError(messageForAuthError(e?.code));
    } finally {
      setSubmitting(false);
    }
  }

  /**
   * サインイン
   * - email と password だけで Firebase Auth にログイン
   * - ★ backend.bootstrap は呼び出さない（プロフィール不要）
   */
  async function signIn(email: string, password: string) {
    setSubmitting(true);
    setError(null);
    try {
      await signInWithEmailAndPassword(auth, email, password);
      // ログイン成功後の処理は、useAuth やルーティング側に任せる
    } catch (e: any) {
      console.error("signIn error", e);
      setError(e?.message ?? "ログインに失敗しました");
    } finally {
      setSubmitting(false);
    }
  }

  async function signOutCurrentUser() {
    setSubmitting(true);
    setError(null);
    try {
      await signOut(auth);
    } catch (e: any) {
      console.error("signOut error", e);
      setError(e?.message ?? "ログアウトに失敗しました");
    } finally {
      setSubmitting(false);
    }
  }

  return { signUp, signIn, signOut: signOutCurrentUser, submitting, error, setError };
}
