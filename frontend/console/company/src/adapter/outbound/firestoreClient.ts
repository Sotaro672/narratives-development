// frontend/console/company/src/adapter/outbound/firestoreClient.ts

import { initializeApp, type FirebaseApp } from "firebase/app";
import { getFirestore, type Firestore } from "firebase/firestore";

/**
 * ------------------------------------------------------------
 *  Firebase App 初期化
 *  （company モジュール専用の Firebase クライアント）
 * ------------------------------------------------------------
 */

// tsconfig で vite/client 型を読んでいないため、
// import.meta.env への型エラーを避けるために any キャストで受ける
const env = (import.meta as any).env as {
  VITE_FIREBASE_API_KEY?: string;
  VITE_FIREBASE_AUTH_DOMAIN?: string;
  VITE_FIREBASE_PROJECT_ID?: string;
  VITE_FIREBASE_STORAGE_BUCKET?: string;
  VITE_FIREBASE_MESSAGING_SENDER_ID?: string;
  VITE_FIREBASE_APP_ID?: string;
  VITE_FIREBASE_MEASUREMENT_ID?: string;
};

const firebaseConfig = {
  apiKey: env.VITE_FIREBASE_API_KEY,
  authDomain: env.VITE_FIREBASE_AUTH_DOMAIN,
  projectId: env.VITE_FIREBASE_PROJECT_ID,
  storageBucket: env.VITE_FIREBASE_STORAGE_BUCKET,
  messagingSenderId: env.VITE_FIREBASE_MESSAGING_SENDER_ID,
  appId: env.VITE_FIREBASE_APP_ID,
  measurementId: env.VITE_FIREBASE_MEASUREMENT_ID,
};

// アプリが重複初期化されないように保持用変数
let firebaseApp: FirebaseApp | null = null;
let dbInstance: Firestore | null = null;

function initialize(): { app: FirebaseApp; db: Firestore } {
  if (!firebaseApp) {
    firebaseApp = initializeApp(firebaseConfig);
  }
  if (!dbInstance) {
    dbInstance = getFirestore(firebaseApp);
  }

  return {
    app: firebaseApp,
    db: dbInstance,
  };
}

/**
 * ------------------------------------------------------------
 * Export: Firestore の db を提供
 * ------------------------------------------------------------
 */
export const { db } = initialize();
