// frontend/member/src/adapter/outbound/firestoreClient.ts
import { initializeApp, getApps, FirebaseApp } from "firebase/app";
import { getFirestore, Firestore } from "firebase/firestore";

/**
 * Vite（もしくはフロントエンドビルド環境）で注入される環境変数を使用します。
 * 例:
 *   VITE_FIREBASE_API_KEY=xxx
 *   VITE_FIREBASE_AUTH_DOMAIN=xxx
 *   VITE_FIREBASE_PROJECT_ID=xxx
 *   VITE_FIREBASE_STORAGE_BUCKET=xxx
 *   VITE_FIREBASE_MESSAGING_SENDER_ID=xxx
 *   VITE_FIREBASE_APP_ID=xxx
 */
const firebaseConfig = {
  apiKey: import.meta.env.VITE_FIREBASE_API_KEY as string,
  authDomain: import.meta.env.VITE_FIREBASE_AUTH_DOMAIN as string,
  projectId: import.meta.env.VITE_FIREBASE_PROJECT_ID as string,
  storageBucket: import.meta.env.VITE_FIREBASE_STORAGE_BUCKET as string,
  messagingSenderId: import.meta.env.VITE_FIREBASE_MESSAGING_SENDER_ID as string,
  appId: import.meta.env.VITE_FIREBASE_APP_ID as string,
};

let firebaseApp: FirebaseApp | null = null;
let firestoreClient: Firestore | null = null;

/**
 * Firebase App をシングルトンで初期化
 */
function getFirebaseApp(): FirebaseApp {
  if (firebaseApp) return firebaseApp;

  if (!firebaseConfig.apiKey || !firebaseConfig.projectId) {
    // 本番では logger に流すなどしてよい
    console.warn(
      "[firestoreClient] Missing Firebase environment variables. Check your .env / VITE_FIREBASE_* settings."
    );
  }

  if (!getApps().length) {
    firebaseApp = initializeApp(firebaseConfig);
  } else {
    firebaseApp = getApps()[0]!;
  }

  return firebaseApp!;
}

/**
 * Firestore クライアントを取得（フロントエンド用）
 */
export function getFirestoreClient(): Firestore {
  if (firestoreClient) return firestoreClient;

  const app = getFirebaseApp();
  firestoreClient = getFirestore(app);
  return firestoreClient;
}
