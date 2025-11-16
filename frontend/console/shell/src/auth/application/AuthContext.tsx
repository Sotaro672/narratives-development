// frontend/console/shell/src/auth/application/AuthContext.tsx
import * as React from "react";
import type { User } from "firebase/auth";
import { onAuthStateChanged } from "firebase/auth";
import { auth, db } from "../config/firebaseClient";
import { doc, getDoc } from "firebase/firestore";
import type { Auth } from "../domain/auth";

type AuthContextValue = {
  user: Auth | null;
  loading: boolean;
};

const AuthContext = React.createContext<AuthContextValue | undefined>(undefined);

function mapFirebaseUserBase(user: User | null): Omit<Auth, "companyId" | "permissions" | "assignedBrands"> | null {
  if (!user) return null;
  return {
    uid: user.uid,
    email: user.email ?? null,
    displayName: user.displayName ?? null,
  };
}

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [state, setState] = React.useState<AuthContextValue>({
    user: null,
    loading: true,
  });

  React.useEffect(() => {
    let active = true;

    const unsub = onAuthStateChanged(auth, async (firebaseUser) => {
      // 未ログイン
      if (!firebaseUser) {
        if (active) setState({ user: null, loading: false });
        return;
      }

      if (active) setState((s) => ({ ...s, loading: true }));

      try {
        const base = mapFirebaseUserBase(firebaseUser)!;

        // users/{uid} から各種属性を取得
        const userRef = doc(db, "users", firebaseUser.uid);
        const snap = await getDoc(userRef);
        const data = snap.exists() ? (snap.data() as any) : {};

        const companyId: string | null = data?.companyId ?? null;
        const permissions: string[] = Array.isArray(data?.permissions) ? data.permissions : [];
        const assignedBrands: string[] = Array.isArray(data?.assignedBrands) ? data.assignedBrands : [];

        const authUser: Auth = {
          ...base,
          companyId,
          permissions,
          assignedBrands,
        };

        if (active) setState({ user: authUser, loading: false });
      } catch (e) {
        console.error("[AuthContext] failed to load user profile:", e);
        const base = mapFirebaseUserBase(firebaseUser)!;
        // 取得に失敗してもログインは継続。空配列でフォールバック
        const fallback: Auth = {
          ...base,
          companyId: null,
          permissions: [],
          assignedBrands: [],
        };
        if (active) setState({ user: fallback, loading: false });
      }
    });

    return () => {
      active = false;
      unsub();
    };
  }, []);

  return <AuthContext.Provider value={state}>{children}</AuthContext.Provider>;
};

export function useAuthContext(): AuthContextValue {
  const ctx = React.useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuthContext must be used within AuthProvider");
  }
  return ctx;
}
