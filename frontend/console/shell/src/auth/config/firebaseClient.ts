// frontend/console/shell/src/auth/config/firebaseClient.ts
/// <reference types="vite/client" />

import { initializeApp, getApps } from "firebase/app";
import { getAuth } from "firebase/auth";
import {
  getFirestore,
  doc,
  getDoc,
  type DocumentData,
} from "firebase/firestore";

const firebaseConfig = {
  apiKey: import.meta.env.VITE_FIREBASE_API_KEY,
  authDomain: import.meta.env.VITE_FIREBASE_AUTH_DOMAIN,
  projectId: import.meta.env.VITE_FIREBASE_PROJECT_ID,
  storageBucket: import.meta.env.VITE_FIREBASE_STORAGE_BUCKET,
  messagingSenderId: import.meta.env.VITE_FIREBASE_MESSAGING_SENDER_ID,
  appId: import.meta.env.VITE_FIREBASE_APP_ID,
  measurementId: import.meta.env.VITE_FIREBASE_MEASUREMENT_ID,
};

// Reuse app if already initialized (for HMR)
const app = getApps().length ? getApps()[0] : initializeApp(firebaseConfig);

// Export Auth / Firestore
export const auth = getAuth(app);
export const db = getFirestore(app);

/**
 * Get company name by document id in "companies" collection.
 * Returns null if not found or name is empty.
 */
export async function getCompanyNameById(id: string): Promise<string | null> {
  const ref = doc(db, "companies", id.trim());
  const snap = await getDoc(ref);
  if (!snap.exists()) return null;
  const data = snap.data() as DocumentData | undefined;
  const name = (data?.name ?? "").toString().trim();
  return name.length > 0 ? name : null;
}
