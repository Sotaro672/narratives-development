//frontend\console\shell\src\auth\application\AuthContext.tsx
import * as React from "react";
import type { User } from "firebase/auth";
import { onAuthStateChanged } from "firebase/auth";
import { doc, getDoc } from "firebase/firestore";
import { auth, db } from "../infrastructure/config/firebaseClient";
import type { Auth } from "../domain/entity/auth";

type AuthContextValue = {
  user: Auth | null;
  loading: boolean;
};

const AuthContext = React.createContext<AuthContextValue | undefined>(undefined);

function mapFirebaseUserBase(
  user: User | null
): Omit<Auth, "companyId" | "permissions" | "assignedBrands"> | null {
  if (!user) return null;

  return {
    uid: user.uid,
    email: user.email ?? null,
    displayName: user.displayName ?? null,
  };
}

async function loadAuthUser(firebaseUser: User): Promise<Auth> {
  const base = mapFirebaseUserBase(firebaseUser);

  if (!base) {
    throw new Error("Failed to map firebase user.");
  }

  try {
    const userRef = doc(db, "users", firebaseUser.uid);
    const snap = await getDoc(userRef);
    const data = snap.exists() ? (snap.data() as Record<string, unknown>) : {};

    const companyId =
      typeof data.companyId === "string" ? data.companyId : null;

    const permissions = Array.isArray(data.permissions)
      ? data.permissions.filter(
          (value): value is string => typeof value === "string"
        )
      : [];

    const assignedBrands = Array.isArray(data.assignedBrands)
      ? data.assignedBrands.filter(
          (value): value is string => typeof value === "string"
        )
      : [];

    return {
      ...base,
      companyId,
      permissions,
      assignedBrands,
    };
  } catch (error) {
    console.error("[AuthContext] failed to load user profile:", error);

    return {
      ...base,
      companyId: null,
      permissions: [],
      assignedBrands: [],
    };
  }
}

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const [state, setState] = React.useState<AuthContextValue>({
    user: null,
    loading: true,
  });

  const initializedRef = React.useRef(false);
  const lastResolvedUidRef = React.useRef<string | null>(null);

  React.useEffect(() => {
    let active = true;

    const unsubscribe = onAuthStateChanged(auth, async (firebaseUser) => {
      if (!active) return;

      if (!initializedRef.current) {
        initializedRef.current = true;
      }

      if (!firebaseUser) {
        lastResolvedUidRef.current = null;
        setState({
          user: null,
          loading: false,
        });
        return;
      }

      if (lastResolvedUidRef.current !== firebaseUser.uid) {
        setState((prev) => ({
          ...prev,
          loading: true,
        }));
      }

      const authUser = await loadAuthUser(firebaseUser);

      if (!active) return;

      lastResolvedUidRef.current = firebaseUser.uid;
      setState({
        user: authUser,
        loading: false,
      });
    });

    return () => {
      active = false;
      unsubscribe();
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