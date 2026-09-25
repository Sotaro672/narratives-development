// frontend/mall/src/lib/authReady.ts

import {
  onAuthStateChanged,
  type User,
} from "firebase/auth";

import { auth } from "./firebase";

let authReadyPromise: Promise<User | null> | null = null;

export function waitForAuthReady(): Promise<User | null> {
  if (auth.currentUser) {
    return Promise.resolve(auth.currentUser);
  }

  if (authReadyPromise) {
    return authReadyPromise;
  }

  authReadyPromise = new Promise<User | null>((resolve, reject) => {
    let unsubscribe = () => {};

    unsubscribe = onAuthStateChanged(
      auth,
      (user) => {
        unsubscribe();
        authReadyPromise = null;
        resolve(user ?? null);
      },
      (error) => {
        unsubscribe();
        authReadyPromise = null;
        reject(error);
      },
    );
  });

  return authReadyPromise;
}