// frontend/console/shell/src/auth/application/profileService.ts
import {
  EmailAuthProvider,
  reauthenticateWithCredential,
  updatePassword as fbUpdatePassword,
  verifyBeforeUpdateEmail,
  sendPasswordResetEmail, // ★ 追加
} from "firebase/auth";
import { auth } from "../infrastructure/config/firebaseClient";

// Firebase エラーコード → アプリ内コード
function mapFirebaseErrorCode(code?: string): string {
  switch (code) {
    // 再認証系
    case "auth/wrong-password":
    case "auth/user-mismatch":
    case "auth/user-not-found":
    case "auth/user-disabled":
    case "auth/invalid-credential":
      return "AUTH_REAUTH_FAILED";

    // メール重複
    case "auth/email-already-in-use":
      return "AUTH_EMAIL_IN_USE";

    // パスワード強度不足
    case "auth/weak-password":
      return "AUTH_WEAK_PASSWORD";

    default:
      return "AUTH_UNKNOWN";
  }
}

// ─────────────────────────────
// メールアドレス変更
// ─────────────────────────────
export async function changeEmail(
  currentPassword: string,
  newEmail: string,
): Promise<void> {
  const user = auth.currentUser;
  if (!user || !user.email) {
    throw new Error("AUTH_NO_USER");
  }

  try {
    const cred = EmailAuthProvider.credential(user.email, currentPassword);
    await reauthenticateWithCredential(user, cred);

    await verifyBeforeUpdateEmail(user, newEmail);
  } catch (e: any) {
    const code = e?.code as string | undefined;
    const mapped = mapFirebaseErrorCode(code);
    throw new Error(mapped);
  }
}

// ─────────────────────────────
// （旧）パスワード変更
//   → 仕様変更により「再設定メール送信」を使うため、
//     直接パスワードを更新する処理は今後は使わない想定。
//   必要であれば残してもよいが、ここでは未使用。
// ─────────────────────────────
export async function changePassword(
  currentPassword: string,
  newPassword: string,
): Promise<void> {
  const user = auth.currentUser;
  if (!user || !user.email) {
    throw new Error("AUTH_NO_USER");
  }

  try {
    const cred = EmailAuthProvider.credential(user.email, currentPassword);
    await reauthenticateWithCredential(user, cred);

    await fbUpdatePassword(user, newPassword);
  } catch (e: any) {
    const code = e?.code as string | undefined;
    const mapped = mapFirebaseErrorCode(code);
    throw new Error(mapped);
  }
}

// ─────────────────────────────
// ★ 追加: パスワード再設定メール送信
//   Firebase のテンプレート
//   「%APP_NAME% のパスワードを再設定してください」メールが送信される
// ─────────────────────────────
export async function sendPasswordResetForCurrentUser(): Promise<void> {
  const user = auth.currentUser;
  if (!user || !user.email) {
    throw new Error("AUTH_NO_USER");
  }

  try {
    await sendPasswordResetEmail(auth, user.email);
  } catch (e: any) {
    const code = e?.code as string | undefined;
    const mapped = mapFirebaseErrorCode(code);
    throw new Error(mapped);
  }
}
