// frontend/src/features/auth/services/createAccountService.ts

import { createUserWithEmailAndPassword, type Auth } from "firebase/auth";

import type { CreateAccountParams, CreateAccountResult } from "../types";
import {
  isEmailValid,
  isPasswordMatch,
  isPasswordValid,
  normalizeEmail,
} from "../utils/authValidation";

type ServiceParams = CreateAccountParams & {
  auth: Auth;
};

function buildVerificationEmailEndpoint(): string {
  const apiBaseURL = import.meta.env.VITE_API_BASE_URL?.replace(/\/+$/, "");

  if (!apiBaseURL) {
    throw new Error("APIの接続先が設定されていません。");
  }

  return `${apiBaseURL}/auth/email-verification/send`;
}

async function requestVerificationEmail(idToken: string): Promise<void> {
  const response = await fetch(buildVerificationEmailEndpoint(), {
    method: "POST",
    headers: {
      Authorization: `Bearer ${idToken}`,
      "Content-Type": "application/json",
      Accept: "application/json",
    },
    body: JSON.stringify({}),
  });

  const contentType = response.headers.get("content-type") || "";

  if (!response.ok) {
    let message = "確認メールの送信に失敗しました。";

    if (contentType.includes("application/json")) {
      try {
        const data = (await response.json()) as {
          error?: string;
          message?: string;
        };

        message = data.error || data.message || message;
      } catch {
        // レスポンスがJSONでない場合は既定メッセージを使う
      }
    }

    throw new Error(message);
  }

  if (!contentType.includes("application/json")) {
    throw new Error(
      "確認メール送信APIではなく、別のページが返されました。APIのURL設定を確認してください。"
    );
  }
}

export async function createAccountAndSendVerification({
  auth,
  emailRaw,
  password,
  passwordConfirmation,
  agree,
}: ServiceParams): Promise<CreateAccountResult> {
  const email = normalizeEmail(emailRaw);

  if (!email) {
    return {
      ok: false,
      error: "メールアドレスを入力してください。",
    };
  }

  if (!isEmailValid(email)) {
    return {
      ok: false,
      error: "メールアドレスの形式が正しくありません。",
    };
  }

  if (!isPasswordValid(password)) {
    return {
      ok: false,
      error: "パスワードは6文字以上で入力してください。",
    };
  }

  if (!isPasswordMatch(password, passwordConfirmation)) {
    return {
      ok: false,
      error: "パスワード確認用が一致していません。",
    };
  }

  if (!agree) {
    return {
      ok: false,
      error: "利用規約に同意してください。",
    };
  }

  try {
    const credential = await createUserWithEmailAndPassword(
      auth,
      email,
      password
    );

    const idToken = await credential.user.getIdToken();

    await requestVerificationEmail(idToken);

    return {
      ok: true,
      email,
    };
  } catch (e) {
    if (e instanceof Error) {
      return {
        ok: false,
        error: e.message,
      };
    }

    return {
      ok: false,
      error: "アカウント作成に失敗しました。",
    };
  }
}