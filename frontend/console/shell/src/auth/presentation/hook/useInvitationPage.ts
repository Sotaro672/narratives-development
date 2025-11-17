// frontend/console/shell/src/auth/presentation/hook/useInvitationPage.ts
import { useCallback, useRef, useState } from "react";

export function useInvitationPage() {
  // ---- フォーム ref ----
  const formRef = useRef<HTMLFormElement>(null);

  // ---- 氏名系 ----
  const [lastName, setLastName] = useState("");
  const [lastNameKana, setLastNameKana] = useState("");
  const [firstName, setFirstName] = useState("");
  const [firstNameKana, setFirstNameKana] = useState("");

  // ---- パスワード ----
  const [password, setPassword] = useState("");
  const [passwordConfirm, setPasswordConfirm] = useState("");

  // ---- 招待トークンから取得する割り当て情報 ----
  // ※ InvitationPage.tsx と同じ初期値
  const [companyId, setCompanyId] = useState<string>("");
  const [assignedBrandIds, setAssignedBrandIds] = useState<string>("");
  const [permissions, setPermissions] = useState<string>("");

  // ---- Navigation ----
  const handleBack = useCallback(() => {
    history.back();
  }, []);

  const handleCreate = useCallback(() => {
    formRef.current?.requestSubmit();
  }, []);

  // ---- Submit ----
  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      // ※このロジックは後で:
      // 1) Firebase auth.createUserWithEmailAndPassword
      // 2) sendEmailVerification
      // 3) バックエンドに「招待完了」通知
      // というフローに差し替える
      console.log({
        lastName,
        lastNameKana,
        firstName,
        firstNameKana,
        password,
        passwordConfirm,
        companyId,
        assignedBrandIds,
        permissions,
      });
    },
    [
      lastName,
      lastNameKana,
      firstName,
      firstNameKana,
      password,
      passwordConfirm,
      companyId,
      assignedBrandIds,
      permissions,
    ],
  );

  // ---- return ----
  return {
    formRef,

    // 氏名
    lastName,
    setLastName,
    lastNameKana,
    setLastNameKana,
    firstName,
    setFirstName,
    firstNameKana,
    setFirstNameKana,

    // パスワード
    password,
    setPassword,
    passwordConfirm,
    setPasswordConfirm,

    // 割り当て情報
    companyId,
    setCompanyId,
    assignedBrandIds,
    setAssignedBrandIds,
    permissions,
    setPermissions,

    // Actions
    handleBack,
    handleCreate,
    handleSubmit,
  };
}
