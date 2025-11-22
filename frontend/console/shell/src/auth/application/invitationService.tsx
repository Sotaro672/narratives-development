// frontend/console/shell/src/auth/application/invitationService.tsx
import {
  createUserWithEmailAndPassword,
  sendEmailVerification,
} from "firebase/auth";
import { auth } from "../infrastructure/config/firebaseClient";

// 🔙 BACKEND の BASE URL
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/g,
    "",
  ) ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";
const API_BASE = ENV_BASE || FALLBACK_BASE;

// ------------------------------
// 型定義
// ------------------------------

export type InvitationInfo = {
  memberId: string;
  companyId: string;
  assignedBrandIds: string[];
  permissions: string[];
  email?: string; // ★ 追加（Firestore には email がある想定なので optional で受ける）
};

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

// validate API の戻り値想定
type ValidateResponse = {
  email: string;
  memberId?: string;
  companyId?: string;
  assignedBrandIds?: string[];
  permissions?: string[];
};

// ------------------------------
// 招待情報取得（GET /api/invitation）
// ------------------------------
export async function fetchInvitationInfo(token: string): Promise<InvitationInfo> {
  const url = `${API_BASE}/api/invitation?token=${encodeURIComponent(token)}`;

  // eslint-disable-next-line no-console
  console.log("[InvitationService] Fetching invitation info:", url);

  const res = await fetch(url, {
    method: "GET",
    headers: {
      "Content-Type": "application/json",
    },
  });

  const text = await res.text();
  // eslint-disable-next-line no-console
  console.log("[InvitationService] raw response:", text);

  if (!res.ok) {
    throw new Error(`Failed to load invitation info (status ${res.status})`);
  }

  // email を含めてパース（無ければ undefined）
  const data = JSON.parse(text) as InvitationInfo;

  return data;
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

  if (!password || !passwordConfirm) {
    throw new Error("パスワードが指定されていません。");
  }
  if (password !== passwordConfirm) {
    throw new Error("パスワードが一致していません。");
  }

  // 1) backend: /api/invitation/validate(token)
  const validateRes = await fetch(`${API_BASE}/api/invitation/validate`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ token }),
  });

  const validateText = await validateRes.text();
  // eslint-disable-next-line no-console
  console.log("[InvitationService] validate response:", validateText);

  if (!validateRes.ok) {
    let msg = `招待の検証に失敗しました (status ${validateRes.status})`;
    try {
      const errJson = JSON.parse(validateText) as { error?: string };
      if (errJson.error) msg = errJson.error;
    } catch {
      // ignore
    }
    throw new Error(msg);
  }

  const validateData = JSON.parse(validateText) as ValidateResponse;

  const email = validateData.email;
  if (!email) {
    throw new Error("招待情報にメールアドレスが含まれていません。");
  }

  const effectiveCompanyId = validateData.companyId ?? companyId;
  const effectiveBrandIds = validateData.assignedBrandIds ?? assignedBrandIds;
  const effectivePermissions = validateData.permissions ?? permissions;

  // 2) Firebase: createUserWithEmailAndPassword
  const cred = await createUserWithEmailAndPassword(auth, email, password);

  // 3) Firebase: sendEmailVerification
  await sendEmailVerification(cred.user);
  // eslint-disable-next-line no-console
  console.log("[InvitationService] verification email sent");

  // 4) backend: /api/invitation/complete(token, uid,...)
  const completePayload = {
    token,
    uid: cred.user.uid,
    profile: {
      lastName,
      lastNameKana,
      firstName,
      firstNameKana,
    },
    companyId: effectiveCompanyId,
    assignedBrandIds: effectiveBrandIds,
    permissions: effectivePermissions,
  };

  // eslint-disable-next-line no-console
  console.log("[InvitationService] complete payload:", completePayload);

  const completeRes = await fetch(`${API_BASE}/api/invitation/complete`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(completePayload),
  });

  const completeText = await completeRes.text();
  // eslint-disable-next-line no-console
  console.log("[InvitationService] complete response:", completeText);

  if (!completeRes.ok) {
    let msg = `招待の完了処理に失敗しました (status ${completeRes.status})`;
    try {
      const errJson = JSON.parse(completeText) as { error?: string };
      if (errJson.error) msg = errJson.error;
    } catch {
      // ignore
    }
    throw new Error(msg);
  }
}
