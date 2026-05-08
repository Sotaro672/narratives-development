// frontend/console/shell/src/auth/infrastructure/query/companyQuery.ts
import { db } from "../config/firebaseClient";
import { doc, getDoc, type DocumentData } from "firebase/firestore";

export async function getCompanyNameById(id: string): Promise<string | null> {
  const ref = doc(db, "companies", id.trim());
  const snap = await getDoc(ref);
  if (!snap.exists()) return null;

  const data = snap.data() as DocumentData | undefined;
  const name = (data?.name ?? "").toString().trim();
  return name.length > 0 ? name : null;
}
