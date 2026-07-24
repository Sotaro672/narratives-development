// frontend/console/shell/src/auth/application/invitationService.tsx
import { createUserWithEmailAndPassword } from "firebase/auth";

import { mapPermissionNamesToDescriptionsJa } from "../../../../permission/src/application/permissionCatalog";
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
  lastName: string;
  lastNameKana: string;
  firstName: string;
  firstNameKana: string;
  password: string;
  passwordConfirm: string;
  companyId: string;
  assignedBrandIds: string[];
  permissions: string[];
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
    lastName,
    lastNameKana,
    firstName,
    firstNameKana,
    password,
    passwordConfirm,
    companyId,
    assignedBrandIds,
    permissions,
  } = params;

  const trimmedToken = token.trim();
  const trimmedLastName = lastName.trim();
  const trimmedLastNameKana = lastNameKana.trim();
  const trimmedFirstName = firstName.trim();
  const trimmedFirstNameKana = firstNameKana.trim();

  if (!trimmedToken) {
    throw new Error("招待トークンが指定されていません。");
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

  // 1. 招待トークンを検証する
  const validateData = await validateInvitation(trimmedToken);

  const email = (validateData.email ?? "").trim();

  if (!email) {
    throw new Error(
      "招待情報にメールアドレスが含まれていません。",
    );
  }

  const effectiveCompanyId =
    validateData.companyId ?? companyId;

  const effectiveBrandIds =
    validateData.assignedBrandIds ?? assignedBrandIds;

  const effectivePermissions =
    validateData.permissions ?? permissions;

  const companyName =
    validateData.companyName ?? effectiveCompanyId;

  const brandNames =
    validateData.brandNames &&
    validateData.brandNames.length > 0
      ? validateData.brandNames
      : effectiveBrandIds;

  const permissionDescriptions =
    mapPermissionNamesToDescriptionsJa(
      effectivePermissions,
    );

  // eslint-disable-next-line no-console
  console.log("[InvitationService] display info:", {
    companyId: effectiveCompanyId,
    companyName,
    brandIds: effectiveBrandIds,
    brandNames,
    permissionNames: effectivePermissions,
    permissionDescriptionsJa: permissionDescriptions,
  });

  // 2. Firebase Authenticationへユーザーを作成する
  // createUserWithEmailAndPassword成功後は、
  // 作成されたユーザーがauth.currentUserになる。
  await createUserWithEmailAndPassword(
    auth,
    email,
    password,
  );

  // 3. Backendで招待を完了する
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