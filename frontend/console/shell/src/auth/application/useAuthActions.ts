// frontend/console/shell/src/auth/application/useAuthActions.ts
import { useState } from "react";
import {
  createUserWithEmailAndPassword,
  signInWithEmailAndPassword,
  signOut,
  deleteUser,
  sendEmailVerification,              // ★ 追加
} from "firebase/auth";
import { auth } from "../infrastructure/config/firebaseClient";

// Firestore
import {
  collection,
  serverTimestamp,
  doc,
  setDoc,
  writeBatch,
  getDoc,
} from "firebase/firestore";
import { db } from "../infrastructure/config/firebaseClient";

// ★ 追加: すべてのPermission定義（新規ユーザーにフル権限付与）
import { ALL_PERMISSIONS } from "../../../../permission/src/infrastructure/mockdata/mockdata";

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

export type SignUpProfile = {
  lastName?: string;
  firstName?: string;
  lastNameKana?: string;
  firstNameKana?: string;
  companyName?: string; // 任意
};

// ★ 追加: 認証メールの actionCodeSettings
// ここで「メールのリンクを踏んだあとに遷移させたいURL」を指定
const actionCodeSettings = {
  url: "https://narratives-development-26c2d.firebaseapp.com/post-verify",
  handleCodeInApp: false,
};

export function useAuthActions() {
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  /** 初期の members/{uid} 作成（companyId: null でOK。あとでバッチで上書き） */
  async function initMember(uid: string, email: string, profile?: SignUpProfile) {
    // ★ 新規ユーザーには全権限を付与
    const initialPermissions = ALL_PERMISSIONS.map((p) => p.name);

    await setDoc(
      doc(db, "members", uid),
      {
        id: uid,
        firstName: (profile?.firstName ?? "").trim(),
        lastName: (profile?.lastName ?? "").trim(),
        firstNameKana: (profile?.firstNameKana ?? "").trim(),
        lastNameKana: (profile?.lastNameKana ?? "").trim(),
        email: email.trim(),
        permissions: initialPermissions,
        assignedBrands: [],
        companyId: null,
        createdAt: serverTimestamp(),
        updatedAt: serverTimestamp(),
      },
      { merge: true },
    );
  }

  /** 会社作成と members.companyId の付与を同一バッチで行う */
  async function createCompanyAndLink(uid: string, companyName?: string | null) {
    const name = (companyName ?? "").trim();
    if (!name) return null;

    // 事前に DocID を採番して、同じバッチで set できるようにする
    const companyRef = doc(collection(db, "companies"));
    const companyId = companyRef.id;

    const memberRef = doc(db, "members", uid);

    const batch = writeBatch(db);

    // companies/{id} 作成
    batch.set(companyRef, {
      id: companyId,
      name,
      admin: uid,
      isActive: true,
      createdAt: serverTimestamp(),
      createdBy: uid,
      updatedAt: serverTimestamp(),
      updatedBy: uid,
      deletedAt: null,
      deletedBy: null,
    });

    // members/{uid} に companyId を同時付与
    batch.set(
      memberRef,
      {
        companyId: companyId,
        updatedAt: serverTimestamp(),
      },
      { merge: true },
    );

    await batch.commit();

    // 念のため検証
    try {
      const snap = await getDoc(memberRef);
      const data = snap.data() as { companyId?: string | null } | undefined;
      if (!data || data.companyId !== companyId) {
        await setDoc(
          memberRef,
          { companyId, updatedAt: serverTimestamp() },
          { merge: true },
        );
      }
    } catch {
      await setDoc(memberRef, { companyId, updatedAt: serverTimestamp() }, { merge: true });
    }

    return companyId;
  }

  /** サインアップ */
  async function signUp(email: string, password: string, profile?: SignUpProfile) {
    setSubmitting(true);
    setError(null);
    try {
      // 1) Authユーザー作成
      const cred = await createUserWithEmailAndPassword(auth, email, password);
      const uid = cred.user?.uid;
      if (!uid) throw new Error("ユーザー作成後に uid を取得できませんでした。");

      // 2) メールアドレス確認メールを送信  ★ここが今回のポイント
      await sendEmailVerification(cred.user, actionCodeSettings);

      // 3) members/{uid} を先に初期化（companyId: null）
      await initMember(uid, email, profile);

      // 4) 会社名があれば、会社作成 + members.companyId を同一バッチで反映
      await createCompanyAndLink(uid, profile?.companyName);
    } catch (e: any) {
      console.error("signUp error", e);
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
