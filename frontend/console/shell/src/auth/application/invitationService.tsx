// frontend/console/shell/src/auth/application/invitationService.tsx
import { createUserWithEmailAndPassword } from "firebase/auth";
import { auth } from "../infrastructure/config/firebaseClient";
import { mapPermissionNamesToDescriptionsJa } from "../../../../permission/src/application/permissionCatalog";

// API 呼び出し系は infra/api に委譲
import {
  fetchInvitationInfo as fetchInvitationInfoApi,
  validateInvitation,
  completeInvitationOnBackend,
} from "../infrastructure/api/invitationApi";
import type { InvitationInfo as InvitationInfoApi } from "../infrastructure/api/invitationApi";

// ------------------------------
// 型定義
// ------------------------------

// API から返る InvitationInfo 型を application からも使えるように re-export
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
// API ラッパー（従来の呼び出し口を維持）
// ------------------------------

export async function fetchInvitationInfo(
  token: string,
): Promise<InvitationInfo> {
  return fetchInvitationInfoApi(token);
}

// ------------------------------
// 招待の完了フロー（Firebase 認証 + backend API）
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

  // 1) backend: POST /invitations/validate
  const validateData = await validateInvitation(trimmedToken);

  const email = (validateData.email ?? "").trim();
  if (!email) {
    throw new Error("招待情報にメールアドレスが含まれていません。");
  }

  const effectiveCompanyId = validateData.companyId ?? companyId;
  const effectiveBrandIds = validateData.assignedBrandIds ?? assignedBrandIds;
  const effectivePermissions = validateData.permissions ?? permissions;

  // 未ログイン招待画面では /companies/{id} / /brands/{id} は認証必須のため呼ばない。
  // 表示名は POST /invitations/validate のレスポンスに含まれる companyName / brandNames を使う。
  const companyName = validateData.companyName ?? effectiveCompanyId;
  const brandNames =
    validateData.brandNames && validateData.brandNames.length > 0
      ? validateData.brandNames
      : effectiveBrandIds;
  const permissionDescriptions =
    mapPermissionNamesToDescriptionsJa(effectivePermissions);

  // eslint-disable-next-line no-console
  console.log("[InvitationService] display info:", {
    companyId: effectiveCompanyId,
    companyName,
    brandIds: effectiveBrandIds,
    brandNames,
    permissionNames: effectivePermissions,
    permissionDescriptionsJa: permissionDescriptions,
  });

  // 2) Firebase: createUserWithEmailAndPassword
  const cred = await createUserWithEmailAndPassword(auth, email, password);

  // 3) backend: POST /invitations/complete
  await completeInvitationOnBackend({
    token: trimmedToken,
    uid: cred.user.uid,
    lastName: trimmedLastName,
    lastNameKana: trimmedLastNameKana,
    firstName: trimmedFirstName,
    firstNameKana: trimmedFirstNameKana,
    email,
  });
}