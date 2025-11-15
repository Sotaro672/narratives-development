// frontend/console/shell/src/auth/application/useAuthActions.ts
import { useState } from "react";
import {
  createUserWithEmailAndPassword,
  signInWithEmailAndPassword,
  signOut,
  deleteUser,
} from "firebase/auth";
import { auth } from "../config/firebaseClient";

// ▼ Firestore 追記分
import {
  doc,
  setDoc,
  serverTimestamp,
} from "firebase/firestore";
import { db } from "../config/firebaseClient";

function messageForAuthError(code?: string): string {
  switch (code) {
    case "auth/admin-restricted-operation":
      return "現在、クライアントからの新規登録が禁止されています。Firebase Console の Authentication 設定で「ユーザー作成の許可」を有効にしてください。";
    case "auth/operation-not-allowed":
      return "Email/Password のサインイン方法が無効です。Firebase Console で有効化してください。";
    case "auth/email-already-in-use":
      return "このメールアドレスは既に登録されています。";
    case "auth/invalid-email":
      return "メールアドレスの形式が正しくありません。";
    case "auth/weak-password":
      return "パスワードが弱すぎます。より強力なパスワードを設定してください。";
    default:
      return "新規登録に失敗しました。設定を確認してください。";
  }
}

export function useAuthActions() {
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function signUp(email: string, password: string) {
    setSubmitting(true);
    setError(null);
    try {
      // 1) Firebase Auth にユーザー作成
      const cred = await createUserWithEmailAndPassword(auth, email, password);
      const uid = cred.user.uid;

      // 2) Firestore の members/{uid} を作成（初期プロフィール）
      //    Member モデルに合わせて最低限の項目を保存します
      const memberRef = doc(db, "members", uid);
      await setDoc(
        memberRef,
        {
          id: uid,
          firstName: "",
          lastName: "",
          firstNameKana: "",
          lastNameKana: "",
          email: email,
          permissions: [],       // 初期は空
          assignedBrands: [],    // 初期は空
          createdAt: serverTimestamp(),
          updatedAt: serverTimestamp(),
          // companyId を使う場合はここで null or 既定値を入れる:
          // companyId: null,
        },
        { merge: true } // 既に存在していた場合も上書きしすぎないようにマージ
      );
    } catch (e: any) {
      console.error("signUp error", e);
      // もし Auth 作成後に Firestore だけ失敗した場合、後続の整合性を保つため削除する場合は下記
      // if (auth.currentUser) { await deleteUser(auth.currentUser).catch(() => {}); }
      setError(messageForAuthError(e?.code));
    } finally {
      setSubmitting(false);
    }
  }

  async function signIn(email: string, password: string) {
    setSubmitting(true);
    setError(null);
    try {
      await signInWithEmailAndPassword(auth, email, password);
    } catch (e: any) {
      console.error("signIn error", e);
      setError(e?.message ?? "ログインに失敗しました");
    } finally {
      setSubmitting(false);
    }
  }

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

  return { signUp, signIn, signOut: signOutCurrentUser, submitting, error, setError };
}
