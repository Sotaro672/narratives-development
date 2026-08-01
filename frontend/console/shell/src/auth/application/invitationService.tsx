// frontend/console/shell/src/auth/application/invitationService.tsx

import {
  createUserWithEmailAndPassword,
} from "firebase/auth";

import {
  auth,
} from "../infrastructure/config/firebaseClient";

import {
  completeInvitationOnBackend,
  validateInvitation,
} from "../infrastructure/repository/invitationRepositoryHTTP";

export {
  fetchInvitationInfo,
} from "../infrastructure/repository/invitationRepositoryHTTP";

export type {
  InvitationInfo,
} from "../infrastructure/repository/invitationRepositoryHTTP";

// ------------------------------
// 型定義
// ------------------------------

export type CompleteInvitationParams = {
  token: string;
  email: string;
  lastName: string;
  lastNameKana: string;
  firstName: string;
  firstNameKana: string;
  password: string;
  passwordConfirm: string;
};

// ------------------------------
// Helpers
// ------------------------------

function normalizeString(
  value: unknown,
): string {
  return typeof value === "string"
    ? value.trim()
    : "";
}

// ------------------------------
// 招待の完了フロー
// ------------------------------

export async function completeInvitation(
  params: CompleteInvitationParams,
): Promise<void> {
  const {
    token,
    email,
    lastName,
    lastNameKana,
    firstName,
    firstNameKana,
    password,
    passwordConfirm,
  } = params;

  const normalizedToken =
    normalizeString(
      token,
    );

  const normalizedEmail =
    normalizeString(
      email,
    ).toLowerCase();

  const normalizedLastName =
    normalizeString(
      lastName,
    );

  const normalizedLastNameKana =
    normalizeString(
      lastNameKana,
    );

  const normalizedFirstName =
    normalizeString(
      firstName,
    );

  const normalizedFirstNameKana =
    normalizeString(
      firstNameKana,
    );

  if (!normalizedToken) {
    throw new Error(
      "招待トークンが指定されていません。",
    );
  }

  if (!normalizedEmail) {
    throw new Error(
      "メールアドレスが指定されていません。",
    );
  }

  if (!normalizedLastName) {
    throw new Error(
      "姓が指定されていません。",
    );
  }

  if (!normalizedLastNameKana) {
    throw new Error(
      "姓（かな）が指定されていません。",
    );
  }

  if (!normalizedFirstName) {
    throw new Error(
      "名が指定されていません。",
    );
  }

  if (!normalizedFirstNameKana) {
    throw new Error(
      "名（かな）が指定されていません。",
    );
  }

  if (
    !password ||
    !passwordConfirm
  ) {
    throw new Error(
      "パスワードが指定されていません。",
    );
  }

  if (
    password !==
    passwordConfirm
  ) {
    throw new Error(
      "パスワードが一致していません。",
    );
  }

  /**
   * 1. 招待トークンの有効性を検証する。
   *
   * validateレスポンスからemail、権限、
   * 内部IDなどの機微情報は取得しない。
   */
  await validateInvitation(
    normalizedToken,
  );

  /**
   * 2. 入力されたメールアドレスで
   * Firebase Authenticationユーザーを作成する。
   *
   * 招待先メールアドレスとの一致は、
   * Backendの招待完了処理で検証する。
   */
  await createUserWithEmailAndPassword(
    auth,
    normalizedEmail,
    password,
  );

  /**
   * 3. Backendで招待を完了する。
   *
   * ID token取得、Authorizationヘッダー設定、
   * 401時のtoken強制更新と再送は、
   * Repositoryから共通fetch処理へ委譲される。
   */
  await completeInvitationOnBackend({
    token:
      normalizedToken,
    lastName:
      normalizedLastName,
    lastNameKana:
      normalizedLastNameKana,
    firstName:
      normalizedFirstName,
    firstNameKana:
      normalizedFirstNameKana,
  });
}