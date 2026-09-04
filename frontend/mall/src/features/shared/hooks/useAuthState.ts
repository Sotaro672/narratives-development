// frontend/amol/src/features/shared/hooks/useAuthState.ts

import {
  useEffect,
  useState,
} from "react";

import {
  onAuthStateChanged,
  type User,
} from "firebase/auth";

import {
  auth,
} from "../../../lib/firebase";

export function useAuthState() {
  const [
    user,
    setUser,
  ] = useState<User | null>(null);

  const [
    authResolved,
    setAuthResolved,
  ] = useState(false);

  useEffect(() => {
    const unsubscribe =
      onAuthStateChanged(
        auth,
        (nextUser) => {
          setUser(nextUser);
          setAuthResolved(true);
        },
        () => {
          setUser(null);
          setAuthResolved(true);
        },
      );

    return unsubscribe;
  }, []);

  return {
    user,
    authResolved,
    isLoggedIn: Boolean(user),
  };
}