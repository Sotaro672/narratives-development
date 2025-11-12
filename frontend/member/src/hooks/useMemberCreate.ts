// frontend/member/src/hooks/useMemberCreate.ts
import { useCallback, useMemo, useState } from "react";
import type { Member, MemberRole } from "../domain/entity/member";
import { MemberRepositoryFS } from "../infrastructure/firestore/memberRepositoryFS";

export type UseMemberCreateOptions = {
  /** 作成成功時に呼ばれます（呼び出し元で navigate などを実施） */
  onSuccess?: (created: Member) => void;
};

export function useMemberCreate(options?: UseMemberCreateOptions) {
  const repo = useMemo(() => new MemberRepositoryFS(), []);

  // ---- フォーム状態 ----
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [firstNameKana, setFirstNameKana] = useState("");
  const [lastNameKana, setLastNameKana] = useState("");
  const [email, setEmail] = useState("");
  const [role, setRole] = useState<MemberRole>("brand-manager");
  const [permissionsText, setPermissionsText] = useState(""); // カンマ区切り
  const [brandsText, setBrandsText] = useState(""); // カンマ区切り

  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const toArray = (s: string) =>
    s
      .split(",")
      .map((x) => x.trim())
      .filter(Boolean);

  const handleSubmit = useCallback(async (e?: React.FormEvent) => {
    e?.preventDefault?.();
    setError(null);
    setSubmitting(true);
    try {
      const id = crypto.randomUUID();
      const now = new Date().toISOString();

      const member: Member = {
        id,
        firstName: firstName.trim() || undefined,
        lastName: lastName.trim() || undefined,
        firstNameKana: firstNameKana.trim() || undefined,
        lastNameKana: lastNameKana.trim() || undefined,
        email: email.trim() || undefined,
        role,
        permissions: toArray(permissionsText),
        assignedBrands: (() => {
          const arr = toArray(brandsText);
          return arr.length ? arr : undefined;
        })(),
        createdAt: now,
        updatedAt: now,
        updatedBy: "console",
        deletedAt: null,
        deletedBy: null,
      };

      const created = await repo.create(member);
      options?.onSuccess?.(created);
    } catch (err: any) {
      setError(err?.message ?? String(err));
    } finally {
      setSubmitting(false);
    }
  }, [
    repo,
    firstName,
    lastName,
    firstNameKana,
    lastNameKana,
    email,
    role,
    permissionsText,
    brandsText,
    options,
  ]);

  return {
    // 値
    firstName, lastName, firstNameKana, lastNameKana, email, role,
    permissionsText, brandsText,
    submitting, error,

    // セッター
    setFirstName, setLastName, setFirstNameKana, setLastNameKana,
    setEmail, setRole, setPermissionsText, setBrandsText,

    // 動作
    handleSubmit,
    setError, // 呼び出し側で明示的に消したい時用
  };
}
