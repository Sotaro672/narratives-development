// frontend/console/shell/src/auth/application/profileService.ts
import {
  EmailAuthProvider,
  reauthenticateWithCredential,
  updatePassword as fbUpdatePassword,
  verifyBeforeUpdateEmail,
} from "firebase/auth";
import { auth } from "../infrastructure/config/firebaseClient";

export async function changeEmail(currentPassword: string, newEmail: string) {
  const user = auth.currentUser;
  if (!user || !user.email) {
    throw new Error("ログイン情報が見つかりません。再ログインしてください。");
  }

  // 1. 再認証
  const cred = EmailAuthProvider.credential(user.email, currentPassword);
  await reauthenticateWithCredential(user, cred);

  // 2. 変更前に「新メールアドレス宛の確認メールを送る」
  await verifyBeforeUpdateEmail(user, newEmail);

  // ※ ここではまだ Auth 上の email は変わりません。
  //    ユーザーが「新メールアドレスに届いたリンクをクリック」した時点で変更されます。
}

export async function changePassword(currentPassword: string, newPassword: string) {
  const user = auth.currentUser;
  if (!user || !user.email) {
    throw new Error("ログイン情報が見つかりません。再ログインしてください。");
  }

  const cred = EmailAuthProvider.credential(user.email, currentPassword);
  await reauthenticateWithCredential(user, cred);

  await fbUpdatePassword(user, newPassword);
}
