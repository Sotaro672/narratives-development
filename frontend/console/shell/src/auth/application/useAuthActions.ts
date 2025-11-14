// frontend/console/shell/src/auth/application/useAuthActions.ts
import { useState } from "react";
import {
  createUserWithEmailAndPassword,
  signInWithEmailAndPassword,
  signOut,
} from "firebase/auth";
import { auth } from "../config/firebaseClient";

export function useAuthActions() {
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // 新規登録（メール + パスワード）
  async function signUp(email: string, password: string) {
    setSubmitting(true);
    setError(null);
    try {
      await createUserWithEmailAndPassword(auth, email, password);
      // 成功すると onAuthStateChanged 経由で user が更新される
    } catch (e: any) {
      console.error("signUp error", e);
      setError(e?.message ?? "新規登録に失敗しました");
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
