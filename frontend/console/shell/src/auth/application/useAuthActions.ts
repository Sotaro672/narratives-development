// frontend/console/shell/src/auth/application/useAuthActions.ts
import { useState } from "react";
import {
  createUserWithEmailAndPassword,
  signInWithEmailAndPassword,
  signOut,
  updateProfile,
  type UserCredential,
} from "firebase/auth";
import { auth } from "../config/firebaseClient";

/** 新規登録時に受け取る追加プロフィール */
export type SignUpProfile = {
  lastName?: string;
  firstName?: string;
  lastNameKana?: string;
  firstNameKana?: string;
  companyName?: string;
};

export function useAuthActions() {
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  /**
   * 新規登録（メール + パスワード + 任意プロフィール）
   * - 第3引数で AuthPage から氏名/かな/会社名を受け取れるように拡張
   * - 氏名が揃っていれば displayName を "姓 名" に更新
   * - 戻り値で userCredential と profile を返却（後段で Firestore などに保存しやすく）
   */
  async function signUp(
    email: string,
    password: string,
    profile?: SignUpProfile
  ): Promise<{ cred: UserCredential; profile?: SignUpProfile }> {
    setSubmitting(true);
    setError(null);
    try {
      const cred = await createUserWithEmailAndPassword(auth, email, password);

      // displayName を "姓 名" に更新（姓/名のどちらかでもあれば設定）
      const ln = profile?.lastName?.trim();
      const fn = profile?.firstName?.trim();
      const displayName =
        ln && fn ? `${ln} ${fn}` : ln ? ln : fn ? fn : undefined;

      if (displayName && cred.user) {
        await updateProfile(cred.user, { displayName });
      }

      return { cred, profile };
    } catch (e: any) {
      console.error("signUp error", e);
      setError(e?.message ?? "新規登録に失敗しました");
      throw e;
    } finally {
      setSubmitting(false);
    }
  }

  // ログイン（メール + パスワード）
  async function signIn(email: string, password: string) {
    setSubmitting(true);
    setError(null);
    try {
      await signInWithEmailAndPassword(auth, email, password);
      // 成功すると onAuthStateChanged 経由で user が更新される
    } catch (e: any) {
      console.error("signIn error", e);
      setError(e?.message ?? "ログインに失敗しました");
      throw e;
    } finally {
      setSubmitting(false);
    }
  }

  // ログアウト（Header などから呼び出し）
  async function signOutCurrentUser() {
    setSubmitting(true);
    setError(null);
    try {
      await signOut(auth);
    } catch (e: any) {
      console.error("signOut error", e);
      setError(e?.message ?? "ログアウトに失敗しました");
      throw e;
    } finally {
      setSubmitting(false);
    }
  }

  return {
    signUp,
    signIn,
    signOut: signOutCurrentUser,
    submitting,
    error,
    setError,
  };
}
