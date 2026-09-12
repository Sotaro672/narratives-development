// frontend/console/shell/src/auth/application/useAuthActions.ts

import {
  useState,
} from "react";

import {
  createUserWithEmailAndPassword,
  signInWithEmailAndPassword,
  signOut,
} from "firebase/auth";

import {
  auth,
} from "../infrastructure/config/firebaseClient";

import {
  API_BASE,
} from "../../shared/http/apiBase";

import {
  fetchJSON,
  HttpError,
} from "../../shared/http/fetchJSON";

/**
 * 認証エラーからFirebaseのエラーコードを取得する。
 */
function getAuthErrorCode(
  error: unknown,
): string | undefined {
  if (
    typeof error !== "object" ||
    error === null ||
    !("code" in error)
  ) {
    return undefined;
  }

  const code =
    error.code;

  return typeof code === "string"
    ? code
    : undefined;
}

/**
 * unknown型のエラーからメッセージを取得する。
 */
function getErrorMessage(
  error: unknown,
  fallback: string,
): string {
  if (
    error instanceof Error &&
    error.message
  ) {
    return error.message;
  }

  if (
    typeof error === "object" &&
    error !== null &&
    "message" in error &&
    typeof error.message === "string" &&
    error.message
  ) {
    return error.message;
  }

  return fallback;
}

/**
 * Backendから返却されたJSONエラー本文から
 * error文字列を取り出す。
 */
function getBackendErrorMessage(
  bodyText?: string,
): string | null {
  if (!bodyText) {
    return null;
  }

  try {
    const parsed =
      JSON.parse(
        bodyText,
      ) as unknown;

    if (
      typeof parsed === "object" &&
      parsed !== null &&
      "error" in parsed &&
      typeof parsed.error === "string" &&
      parsed.error.trim()
    ) {
      return parsed.error.trim();
    }
  } catch {
    // JSONでない場合は下でプレーンテキストとして扱う。
  }

  const normalized =
    bodyText.trim();

  return normalized || null;
}

/**
 * Bootstrap API失敗時に
 * ユーザーへ表示するメッセージを生成する。
 */
function messageForBootstrapError(
  error: unknown,
  fallback: string,
): string {
  if (error instanceof HttpError) {
    const backendMessage =
      getBackendErrorMessage(
        error.bodyText,
      );

    if (backendMessage) {
      return `${fallback} ${backendMessage}`;
    }

    return `${fallback} HTTP ${error.status}`;
  }

  return getErrorMessage(
    error,
    fallback,
  );
}

/**
 * 新規登録時の認証エラーメッセージ。
 */
function messageForSignUpError(
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

    case "auth/missing-password":
      return "パスワードを入力してください。";

    case "auth/network-request-failed":
      return "通信に失敗しました。ネットワーク接続を確認してください。";

    case "auth/too-many-requests":
      return "短時間に多数の操作が行われました。時間を置いて再度お試しください。";

    default:
      return "新規登録に失敗しました。設定を確認してください。";
  }
}

/**
 * ログイン時の認証エラーメッセージ。
 */
function messageForSignInError(
  code?: string,
): string {
  switch (code) {
    case "auth/invalid-email":
      return "メールアドレスの形式が正しくありません。";

    case "auth/missing-password":
      return "パスワードを入力してください。";

    case "auth/invalid-credential":
    case "auth/user-not-found":
    case "auth/wrong-password":
      return "メールアドレスまたはパスワードが正しくありません。";

    case "auth/user-disabled":
      return "このアカウントは無効化されています。";

    case "auth/network-request-failed":
      return "通信に失敗しました。ネットワーク接続を確認してください。";

    case "auth/too-many-requests":
      return "短時間に多数のログイン操作が行われました。時間を置いて再度お試しください。";

    case "auth/operation-not-allowed":
      return "Email/Password のサインイン方法が無効です。Firebase Console で有効化してください。";

    default:
      return "ログインに失敗しました。";
  }
}

/**
 * 値を前後空白除去済みの文字列へ変換する。
 */
function normalizeString(
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

const BOOTSTRAP_URL =
  `${API_BASE}/auth/bootstrap`;

/**
 * サーバーへ送信するprofile bodyを、
 * 空文字を含めない形式で組み立てる。
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
    normalizeString(
      profile?.lastName,
    );

  const firstName =
    normalizeString(
      profile?.firstName,
    );

  const lastNameKana =
    normalizeString(
      profile?.lastNameKana,
    );

  const firstNameKana =
    normalizeString(
      profile?.firstNameKana,
    );

  const companyName =
    normalizeString(
      profile?.companyName,
    );

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
 * BackendのBootstrap APIを呼び出す。
 *
 * ID tokenの取得、Authorizationヘッダーの設定、
 * 401時のtoken強制更新と1回限りの再送は
 * shared/http/fetchJSON.tsへ委譲する。
 *
 * Backend側ではMemberとCompanyが
 * 冪等に作成される前提。
 *
 * Bootstrapに失敗した場合は例外を呼び出し元へ伝播させる。
 */
async function callBootstrap(
  profile?: SignUpProfile,
): Promise<void> {
  const body =
    buildBootstrapBody(
      profile,
    );

  await fetchJSON<void>(
    BOOTSTRAP_URL,
    {
      method: "POST",
      mode: "cors",
      auth: "required",
      headers: {
        "Content-Type":
          "application/json",
      },
      body: JSON.stringify(
        body,
      ),
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
    normalizeString(
      profile?.lastName,
    );

  const firstName =
    normalizeString(
      profile?.firstName,
    );

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
   *
   * - Firebase Authenticationでユーザーを作成する
   * - 作成後にBackendのbootstrapを呼び出す
   * - Bootstrap失敗時はエラーを握りつぶさず画面へ通知する
   */
  async function signUp(
    email: string,
    password: string,
    profile?: SignUpProfile,
  ): Promise<void> {
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

      if (!credential.user?.uid) {
        throw new Error(
          "ユーザー作成後にuidを取得できませんでした。",
        );
      }

      await callBootstrap(
        profile,
      );
    } catch (caughtError: unknown) {
      const code =
        getAuthErrorCode(
          caughtError,
        );

      if (code) {
        setError(
          messageForSignUpError(
            code,
          ),
        );
      } else {
        setError(
          messageForBootstrapError(
            caughtError,
            "アカウント情報の作成に失敗しました。",
          ),
        );
      }

      console.error(
        "[useAuthActions] signUp error:",
        caughtError,
      );
    } finally {
      setSubmitting(false);
    }
  }

  /**
   * サインイン
   *
   * - EmailとPasswordでログインする
   * - ログイン成功後にBackendのbootstrapを呼び出す
   * - Bootstrap失敗時はエラーを握りつぶさず画面へ通知する
   */
  async function signIn(
    email: string,
    password: string,
    profile?: SignUpProfile,
  ): Promise<void> {
    setSubmitting(true);
    setError(null);

    try {
      await signInWithEmailAndPassword(
        auth,
        email,
        password,
      );

      await callBootstrap(
        profile,
      );
    } catch (caughtError: unknown) {
      const code =
        getAuthErrorCode(
          caughtError,
        );

      if (code) {
        setError(
          messageForSignInError(
            code,
          ),
        );
      } else {
        setError(
          messageForBootstrapError(
            caughtError,
            "ログイン後のアカウント情報確認に失敗しました。",
          ),
        );
      }

      console.error(
        "[useAuthActions] signIn error:",
        caughtError,
      );
    } finally {
      setSubmitting(false);
    }
  }

  /**
   * 現在のFirebaseユーザーをログアウトする。
   */
  async function signOutCurrentUser(): Promise<void> {
    setSubmitting(true);
    setError(null);

    try {
      await signOut(
        auth,
      );
    } catch (caughtError: unknown) {
      setError(
        getErrorMessage(
          caughtError,
          "ログアウトに失敗しました。",
        ),
      );

      console.error(
        "[useAuthActions] signOut error:",
        caughtError,
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