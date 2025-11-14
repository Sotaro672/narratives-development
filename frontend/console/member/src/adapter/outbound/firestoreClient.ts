// frontend/console/member/src/adapter/outbound/firestoreClient.ts
import { initializeApp, getApps, FirebaseApp } from "firebase/app";
import { getFirestore, Firestore } from "firebase/firestore";

const env = import.meta.env;

// .env（VITE_）優先＋既知値のフォールバック
const firebaseConfig = {
  apiKey: env.VITE_FIREBASE_API_KEY || "AIzaSyDTetB8PcVlSHhXbItMZv2thd5lY4d5nIQ",
  authDomain: env.VITE_FIREBASE_AUTH_DOMAIN || "narratives-development-26c2d.firebaseapp.com",
  projectId: env.VITE_FIREBASE_PROJECT_ID || "narratives-development-26c2d",
  storageBucket: env.VITE_FIREBASE_STORAGE_BUCKET || "narratives-development-26c2d.firebasestorage.app",
  messagingSenderId: env.VITE_FIREBASE_MESSAGING_SENDER_ID || "871263659099",
  appId: env.VITE_FIREBASE_APP_ID || "1:871263659099:web:0d4bbdc36e59d7ed8d4b7e",
  // measurementId は不要なら未指定でもOK
  measurementId: env.VITE_FIREBASE_MEASUREMENT_ID || "G-T77JW1DF4V",
};

let firebaseApp: FirebaseApp | null = null;
let firestoreClient: Firestore | null = null;

function getFirebaseApp(): FirebaseApp {
  if (firebaseApp) return firebaseApp;

  // すべて揃っていない時だけ軽い警告（ただしフォールバックで動く）
  const requiredKeys = ["apiKey","authDomain","projectId","appId"];
  const missing = requiredKeys.filter(k => !(firebaseConfig as any)[k]);
  if (missing.length) {
    console.warn(
      `[firestoreClient] Firebase config missing keys: ${missing.join(", ")}. Falling back to defaults.`
    );
  }

  if (!getApps().length) {
    firebaseApp = initializeApp(firebaseConfig);
  } else {
    firebaseApp = getApps()[0]!;
  }
  return firebaseApp!;
}

export function getFirestoreClient(): Firestore {
  if (firestoreClient) return firestoreClient;
  const app = getFirebaseApp();
  firestoreClient = getFirestore(app);
  return firestoreClient;
}
