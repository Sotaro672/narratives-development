//frontend\console\shell\src\auth\application\invitationService.tsx
import { createUserWithEmailAndPassword } from "firebase/auth";
import { auth } from "../infrastructure/config/firebaseClient";
import { mapPermissionNamesToDescriptionsJa } from "../../../../permission/src/application/permissionCatalog";

// API 呼び出し系は infra/api に委譲
import {
  fetchInvitationInfo as fetchInvitationInfoApi,
  fetchCompanyNameById as fetchCompanyNameByIdApi,
  fetchBrandNamesByIds as fetchBrandNamesByIdsApi,
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

export { fetchCompanyNameByIdApi as fetchCompanyNameById };
export { fetchBrandNamesByIdsApi as fetchBrandNamesByIds };

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

  if (!token.trim()) {
    throw new Error("招待トークンが指定されていません。");
  }
  if (!lastName.trim()) {
    throw new Error("姓が指定されていません。");
  }
  if (!lastNameKana.trim()) {
    throw new Error("姓（かな）が指定されていません。");
  }
  if (!firstName.trim()) {
    throw new Error("名が指定されていません。");
  }
  if (!firstNameKana.trim()) {
    throw new Error("名（かな）が指定されていません。");
  }
  if (!password || !passwordConfirm) {
    throw new Error("パスワードが指定されていません。");
  }
  if (password !== passwordConfirm) {
    throw new Error("パスワードが一致していません。");
  }

  // 1) backend: /api/invitation/validate(token)
  const validateData = await validateInvitation(token);

  const email = (validateData.email ?? "").trim();
  if (!email) {
    throw new Error("招待情報にメールアドレスが含まれていません。");
  }

  const effectiveCompanyId = validateData.companyId ?? companyId;
  const effectiveBrandIds = validateData.assignedBrandIds ?? assignedBrandIds;
  const effectivePermissions = validateData.permissions ?? permissions;

  // 表示用の名前解決（失敗しても続行）
  try {
    const [companyName, brandNames, permissionDescriptions] = await Promise.all([
      fetchCompanyNameByIdApi(effectiveCompanyId),
      fetchBrandNamesByIdsApi(effectiveBrandIds),
      Promise.resolve(mapPermissionNamesToDescriptionsJa(effectivePermissions)),
    ]);

    // eslint-disable-next-line no-console
    console.log("[InvitationService] display info:", {
      companyId: effectiveCompanyId,
      companyName,
      brandIds: effectiveBrandIds,
      brandNames,
      permissionNames: effectivePermissions,
      permissionDescriptionsJa: permissionDescriptions,
    });
  } catch (e) {
    // eslint-disable-next-line no-console
    console.warn("[InvitationService] failed to resolve display names", e);
  }

  // 2) Firebase: createUserWithEmailAndPassword
  const cred = await createUserWithEmailAndPassword(auth, email, password);

  // 3) backend: /api/invitation/complete(token, uid, 氏名, email)
  await completeInvitationOnBackend({
    token,
    uid: cred.user.uid,
    lastName,
    lastNameKana,
    firstName,
    firstNameKana,
    email,
  });
}