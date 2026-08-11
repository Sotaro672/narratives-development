//services\solana-bubblegum\src\infrastructure\firestore\firestore-client.ts
import {
  Firestore,
} from "@google-cloud/firestore";

export const firestore =
  new Firestore();