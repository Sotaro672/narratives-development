// frontend/console/shell/src/auth/application/invitationService.tsx
import {
  createUserWithEmailAndPassword,
  sendEmailVerification,
} from "firebase/auth";
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
import type {
  InvitationInfo as InvitationInfoApi,
} from "../infrastructure/api/invitationApi";

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

// hook/useInvitationPage など既存コードからは、
// これまで通り invitationService.fetchInvitationInfo を呼べるようにする。
export async function fetchInvitationInfo(token: string): Promise<InvitationInfo> {
  return fetchInvitationInfoApi(token);
}

// companyName / brandNames の解決は presentation からも使いたいので re-export
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

  if (!password || !passwordConfirm) {
    throw new Error("パスワードが指定されていません。");
  }
  if (password !== passwordConfirm) {
    throw new Error("パスワードが一致していません。");
  }

  // 1) backend: /api/invitation/validate(token)
  const validateData = await validateInvitation(token);

  const email = validateData.email;
  if (!email) {
    throw new Error("招待情報にメールアドレスが含まれていません。");
  }

  const effectiveCompanyId = validateData.companyId ?? companyId;
  const effectiveBrandIds = validateData.assignedBrandIds ?? assignedBrandIds;
  const effectivePermissions = validateData.permissions ?? permissions;

  // ★ ここで「id は維持したまま」表示用の名前を取得する
  //    - state や backend payload は ID / permission name のまま
  //    - UI 表示やログで companyName / brandNames / 日本語権限名を使う
  try {
    const [companyName, brandNames, permissionDescriptions] = await Promise.all([
      fetchCompanyNameByIdApi(effectiveCompanyId),
      fetchBrandNamesByIdsApi(effectiveBrandIds),
      Promise.resolve(
        mapPermissionNamesToDescriptionsJa(effectivePermissions),
      ),
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
    // 名前解決に失敗しても、招待完了処理自体は続行したいのでログのみ
    // eslint-disable-next-line no-console
    console.warn("[InvitationService] failed to resolve display names", e);
  }

  // 2) Firebase: createUserWithEmailAndPassword
  const cred = await createUserWithEmailAndPassword(auth, email, password);

  // 3) Firebase: sendEmailVerification
  await sendEmailVerification(cred.user);
  // eslint-disable-next-line no-console
  console.log("[InvitationService] verification email sent");

  // 4) backend: /api/invitation/complete(token, uid,...)
  await completeInvitationOnBackend({
    token,
    uid: cred.user.uid,
    profile: {
      lastName,
      lastNameKana,
      firstName,
      firstNameKana,
    },
    companyId: effectiveCompanyId,       // ← ID をそのまま維持
    assignedBrandIds: effectiveBrandIds, // ← ID をそのまま維持
    permissions: effectivePermissions,   // ← permission name をそのまま維持
  });
}
