// frontend/console/shell/src/auth/application/useAuthActions.ts

import { useState } from "react";
import {
  createUserWithEmailAndPassword,
  signInWithEmailAndPassword,
  signOut,
} from "firebase/auth";

import {
  auth,
} from "../infrastructure/config/firebaseClient";

/**
 * 認証エラーメッセージ
 */
function messageForAuthError(
  code?: string,
): string {
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

function s(
  value: unknown,
): string {
  return String(
    value ?? "",
  ).trim();
}

export type SignUpProfile = {
  lastName?: string;
  firstName?: string;
  lastNameKana?: string;
  firstNameKana?: string;
  companyName?: string;
};

// ─────────────────────────────────────────────
// Backend base URL
// ─────────────────────────────────────────────

const RAW_ENV_BASE =
  (
    (import.meta as any).env
      ?.VITE_BACKEND_BASE_URL as
      | string
      | undefined
  ) ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

function sanitizeBase(
  url: string,
): string {
  return (
    url || ""
  ).replace(
    /\/+$/g,
    "",
  );
}

const ENV_BASE =
  sanitizeBase(
    RAW_ENV_BASE,
  );

const FINAL_BASE =
  sanitizeBase(
    ENV_BASE ||
      FALLBACK_BASE,
  );

if (!FINAL_BASE) {
  throw new Error(
    "[useAuthActions] BACKEND BASE URL is empty. Set VITE_BACKEND_BASE_URL in .env.local",
  );
}

// Backend bootstrap endpoint
const BOOTSTRAP_URL =
  `${FINAL_BASE}/auth/bootstrap`;

// 共通HTTPラッパ
async function httpRequest<T>(
  input: string,
  init: RequestInit = {},
): Promise<T> {
  const response = await fetch(
    input,
    {
      mode: "cors",
      ...init,
      headers: {
        "Content-Type":
          "application/json",
        ...(init.headers ?? {}),
      },
    },
  );

  if (response.status === 204) {
    return undefined as unknown as T;
  }

  const text =
    await response
      .text()
      .catch(() => "");

  if (!response.ok) {
    throw new Error(
      `[useAuthActions] ${response.status} ${response.statusText} :: ${text.slice(0, 300)}`,
    );
  }

  try {
    return text
      ? JSON.parse(text) as T
      : undefined as unknown as T;
  } catch {
    throw new Error(
      `[useAuthActions] JSON parse error. head: ${text.slice(0, 120)}`,
    );
  }
}

/**
 * サーバーに送るprofile bodyを、
 * 空文字を含めない形で組み立てる。
 *
 * Backendが*stringでvalidationしている場合に、
 * 空文字による上書きで
 * "member: invalid firstName"が発生するのを防ぐ。
 */
function buildBootstrapBody(
  profile?: SignUpProfile,
): Record<string, string> {
  const body:
    Record<string, string> = {};

  const lastName =
    s(profile?.lastName);

  const firstName =
    s(profile?.firstName);

  const lastNameKana =
    s(profile?.lastNameKana);

  const firstNameKana =
    s(profile?.firstNameKana);

  const companyName =
    s(profile?.companyName);

  if (lastName) {
    body.lastName =
      lastName;
  }

  if (firstName) {
    body.firstName =
      firstName;
  }

  if (lastNameKana) {
    body.lastNameKana =
      lastNameKana;
  }

  if (firstNameKana) {
    body.firstNameKana =
      firstNameKana;
  }

  if (companyName) {
    body.companyName =
      companyName;
  }

  return body;
}

/**
 * Bootstrap APIを呼び出す。
 *
 * BackendにMemberとCompanyの作成を委譲する。
 * Backend側では冪等に処理される前提。
 */
async function callBootstrap(
  profile?: SignUpProfile,
): Promise<void> {
  const token =
    await auth.currentUser
      ?.getIdToken();

  if (!token) {
    throw new Error(
      "[useAuthActions] Not authenticated (no ID token).",
    );
  }

  const body =
    buildBootstrapBody(
      profile,
    );

  await httpRequest<void>(
    BOOTSTRAP_URL,
    {
      method: "POST",
      body: JSON.stringify(
        body,
      ),
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
    },
  );
}

/**
 * サインアップ前に入力内容を検証する。
 *
 * 不完全な入力でFirebase Authenticationに
 * ユーザーが作成されることを防ぐ。
 */
function validateProfileForSignUp(
  profile?: SignUpProfile,
): string | null {
  const lastName =
    s(profile?.lastName);

  const firstName =
    s(profile?.firstName);

  if (
    !lastName ||
    !firstName
  ) {
    return "姓・名を入力してください。";
  }

  return null;
}

export function useAuthActions() {
  const [
    submitting,
    setSubmitting,
  ] = useState(false);

  const [
    error,
    setError,
  ] = useState<
    string | null
  >(null);

  /**
   * サインアップ
   * - Firebase Authenticationでユーザーを作成
   * - 作成後にBackendのbootstrapを呼び出す
   */
  async function signUp(
    email: string,
    password: string,
    profile?: SignUpProfile,
  ) {
    setSubmitting(true);
    setError(null);

    const validationError =
      validateProfileForSignUp(
        profile,
      );

    if (validationError) {
      setError(
        validationError,
      );

      setSubmitting(false);
      return;
    }

    try {
      const credential =
        await createUserWithEmailAndPassword(
          auth,
          email,
          password,
        );

      const user =
        credential.user;

      if (!user?.uid) {
        throw new Error(
          "ユーザー作成後に uid を取得できませんでした。",
        );
      }

      try {
        await callBootstrap(
          profile,
        );
      } catch {
        // 新規登録自体は成功しているため、
        // bootstrapの失敗は致命的エラーとして扱わない。
      }
    } catch (error: any) {
      setError(
        messageForAuthError(
          error?.code,
        ),
      );
    } finally {
      setSubmitting(false);
    }
  }

  /**
   * サインイン
   * - EmailとPasswordでログイン
   * - ログイン成功後にBackendのbootstrapを呼び出す
   */
  async function signIn(
    email: string,
    password: string,
    profile?: SignUpProfile,
  ) {
    setSubmitting(true);
    setError(null);

    try {
      await signInWithEmailAndPassword(
        auth,
        email,
        password,
      );

      try {
        await callBootstrap(
          profile,
        );
      } catch {
        // ログイン自体は成功しているため、
        // bootstrapの失敗は致命的エラーとして扱わない。
      }
    } catch (error: any) {
      setError(
        error?.message ??
          "ログインに失敗しました",
      );
    } finally {
      setSubmitting(false);
    }
  }

  async function signOutCurrentUser() {
    setSubmitting(true);
    setError(null);

    try {
      await signOut(auth);
    } catch (error: any) {
      setError(
        error?.message ??
          "ログアウトに失敗しました",
      );
    } finally {
      setSubmitting(false);
    }
  }

  return {
    signUp,
    signIn,
    signOut:
      signOutCurrentUser,
    submitting,
    error,
    setError,
  };
}