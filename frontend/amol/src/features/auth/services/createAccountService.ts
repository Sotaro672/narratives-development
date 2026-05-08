// frontend/src/features/auth/services/createAccountService.ts

import {
  createUserWithEmailAndPassword,
  sendEmailVerification,
  type Auth,
} from "firebase/auth";

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

    await sendEmailVerification(credential.user);

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