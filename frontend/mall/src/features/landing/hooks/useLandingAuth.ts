// frontend/mall/src/features/landing/hooks/useLandingAuth.ts

import { useEffect, useState } from "react";
import {
  onAuthStateChanged,
  type User,
} from "firebase/auth";

import { auth } from "../../../lib/firebase";

export type UseLandingAuthResult = {
  currentUser: User | null;
  authResolved: boolean;
  isLoggedIn: boolean;
};

export function useLandingAuth(): UseLandingAuthResult {
  const [currentUser, setCurrentUser] =
    useState<User | null>(null);

  const [authResolved, setAuthResolved] =
    useState(false);

  useEffect(() => {
    const unsubscribe = onAuthStateChanged(
      auth,
      (user) => {
        setCurrentUser(user);
        setAuthResolved(true);
      },
    );

    return unsubscribe;
  }, []);

  return {
    currentUser,
    authResolved,
    isLoggedIn: currentUser !== null,
  };
}

export default useLandingAuth;