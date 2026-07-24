// frontend/console/shell/src/auth/application/invitationService.tsx
import { createUserWithEmailAndPassword } from "firebase/auth";
import {
  completeInvitationOnBackend,
  fetchInvitationInfo as fetchInvitationInfoApi,
  validateInvitation,
} from "../infrastructure/repository/invitationRepositoryHTTP";
import type { InvitationInfo as InvitationInfoApi } from "../infrastructure/repository/invitationRepositoryHTTP";
import { auth } from "../infrastructure/config/firebaseClient";
// ------------------------------
// 型定義
// ------------------------------
export type InvitationInfo = InvitationInfoApi;
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
// APIラッパー
// ------------------------------
export async function fetchInvitationInfo(
  token: string,
): Promise<InvitationInfo> {
  return fetchInvitationInfoApi(token);
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
  const trimmedToken = token.trim();
  const normalizedEmail = email.trim().toLowerCase();
  const trimmedLastName = lastName.trim();
  const trimmedLastNameKana = lastNameKana.trim();
  const trimmedFirstName = firstName.trim();
  const trimmedFirstNameKana = firstNameKana.trim();
  if (!trimmedToken) {
    throw new Error("招待トークンが指定されていません。");
  }
  if (!normalizedEmail) {
    throw new Error("メールアドレスが指定されていません。");
  }
  if (!trimmedLastName) {
    throw new Error("姓が指定されていません。");
  }
  if (!trimmedLastNameKana) {
    throw new Error("姓（かな）が指定されていません。");
  }
  if (!trimmedFirstName) {
    throw new Error("名が指定されていません。");
  }
  if (!trimmedFirstNameKana) {
    throw new Error("名（かな）が指定されていません。");
  }
  if (!password || !passwordConfirm) {
    throw new Error("パスワードが指定されていません。");
  }
  if (password !== passwordConfirm) {
    throw new Error("パスワードが一致していません。");
  }
  // 1. 招待トークンの有効性だけを検証する。
  // validateレスポンスからemail、権限、内部IDは取得しない。
  await validateInvitation(trimmedToken);
  // 2. 入力されたメールアドレスでFirebase Authenticationユーザーを作成する。
  // 招待先メールアドレスとの一致はBackendの招待完了transactionで検証する。
  await createUserWithEmailAndPassword(
    auth,
    normalizedEmail,
    password,
  );
  // 3. Backendで招待を完了する。
  // Repositoryがauth.currentUserからID tokenを取得して
  // Authorizationヘッダーへ設定する。
  await completeInvitationOnBackend({
    token: trimmedToken,
    lastName: trimmedLastName,
    lastNameKana: trimmedLastNameKana,
    firstName: trimmedFirstName,
    firstNameKana: trimmedFirstNameKana,
  });
}