// frontend/console/shell/src/auth/application/AuthContext.tsx
import * as React from "react";
import type { User } from "firebase/auth";
import { onAuthStateChanged } from "firebase/auth";

import { auth } from "../config/firebaseClient";
import type { AuthUser } from "../domain/authUser";

type AuthContextValue = {
  user: AuthUser | null;
  loading: boolean;
};

const AuthContext = React.createContext<AuthContextValue | undefined>(
  undefined,
);

function mapFirebaseUser(user: User | null): AuthUser | null {
  if (!user) return null;
  return {
    uid: user.uid,
    email: user.email ?? null,
    displayName: user.displayName ?? null,
  };
}

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const [state, setState] = React.useState<AuthContextValue>({
    user: null,
    loading: true,
  });

  React.useEffect(() => {
    const unsub = onAuthStateChanged(auth, (firebaseUser) => {
      setState({
        user: mapFirebaseUser(firebaseUser),
        loading: false,
      });
    });

    return () => unsub();
  }, []);

  return (
    <AuthContext.Provider value={state}>{children}</AuthContext.Provider>
  );
};

export function useAuthContext(): AuthContextValue {
  const ctx = React.useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuthContext must be used within AuthProvider");
  }
  return ctx;
}
