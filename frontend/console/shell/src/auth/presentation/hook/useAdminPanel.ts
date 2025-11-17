// frontend/console/shell/src/auth/presentation/hook/useAdminPanel.ts
import { useEffect, useState, useCallback } from "react";
import { auth } from "../../infrastructure/config/firebaseClient";
import {
  fetchCurrentMember,
  updateCurrentMemberProfile,
} from "../../application/memberService";
import {
  changeEmail,
  changePassword,
} from "../../application/profileService";

export function useAdminPanel() {
  // dialog flags
  const [showProfileDialog, setShowProfileDialog] = useState(false);
  const [showEmailDialog, setShowEmailDialog] = useState(false);
  const [showPasswordDialog, setShowPasswordDialog] = useState(false);

  // ★ currentMember の id
  const [memberId, setMemberId] = useState<string | null>(null);

  // profile fields
  const [lastName, setLastName] = useState("");
  const [firstName, setFirstName] = useState("");
  const [lastNameKana, setLastNameKana] = useState("");
  const [firstNameKana, setFirstNameKana] = useState("");

  // email fields
  const [newEmail, setNewEmail] = useState("");
  const [currentPasswordForEmail, setCurrentPasswordForEmail] = useState("");

  // password fields
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");

  // ─────────────────────────────
  // currentMember を取得してフィールドにセット
  // ─────────────────────────────
  useEffect(() => {
    async function loadCurrentMember() {
      const uid = auth.currentUser?.uid;
      if (!uid) return;

      try {
        const member = await fetchCurrentMember(uid);
        if (!member) return;

        setMemberId(member.id ?? uid);

        setFirstName(member.firstName ?? "");
        setLastName(member.lastName ?? "");
        setFirstNameKana(member.firstNameKana ?? "");
        setLastNameKana(member.lastNameKana ?? "");
      } catch (e) {
        console.error("[useAdminPanel] failed to load currentMember:", e);
      }
    }

    void loadCurrentMember();
  }, []);

  // ─────────────────────────────
  // プロフィール保存（Backend 経由で Firestore members を更新）
  // ─────────────────────────────
  const saveProfile = useCallback(async () => {
    if (!memberId) {
      console.warn("[useAdminPanel] memberId is missing, skip saveProfile");
      return;
    }

    try {
      await updateCurrentMemberProfile({
        id: memberId,
        firstName,
        lastName,
        firstNameKana,
        lastNameKana,
      });
      setShowProfileDialog(false);
    } catch (e) {
      console.error("[useAdminPanel] failed to update profile:", e);
    }
  }, [memberId, firstName, lastName, firstNameKana, lastNameKana]);

  // ─────────────────────────────
  // ★ メールアドレス変更 + Firebase メール送信トリガ
  //   （profileService.changeEmail 内で Firebase Auth API を呼び、
  //    コンソールで設定した「メールアドレスの変更」テンプレートが送信される想定）
  // ─────────────────────────────
  const saveEmail = useCallback(async () => {
    if (!newEmail.trim()) {
      throw new Error("EMAIL_REQUIRED");
    }
    if (!currentPasswordForEmail) {
      throw new Error("PASSWORD_REQUIRED");
    }

    await changeEmail(currentPasswordForEmail, newEmail.trim());

    // 成功後は入力をクリア（ダイアログを閉じるかどうかは呼び出し側で制御）
    setNewEmail("");
    setCurrentPasswordForEmail("");
  }, [newEmail, currentPasswordForEmail]);

  // ─────────────────────────────
  // ★ パスワード変更（Firebase Auth + 再認証）
  // ─────────────────────────────
  const savePassword = useCallback(async () => {
    if (!currentPassword) {
      throw new Error("CURRENT_PASSWORD_REQUIRED");
    }
    if (!newPassword) {
      throw new Error("NEW_PASSWORD_REQUIRED");
    }
    if (newPassword !== confirmPassword) {
      throw new Error("PASSWORD_MISMATCH");
    }

    await changePassword(currentPassword, newPassword);

    // 成功後は入力をクリア
    setCurrentPassword("");
    setNewPassword("");
    setConfirmPassword("");
  }, [currentPassword, newPassword, confirmPassword]);

  return {
    // dialog flags
    showProfileDialog,
    setShowProfileDialog,
    showEmailDialog,
    setShowEmailDialog,
    showPasswordDialog,
    setShowPasswordDialog,

    // profile fields
    memberId,
    lastName,
    setLastName,
    lastNameKana,
    setLastNameKana,
    firstName,
    setFirstName,
    firstNameKana,
    setFirstNameKana,

    // email fields
    newEmail,
    setNewEmail,
    currentPasswordForEmail,
    setCurrentPasswordForEmail,

    // password fields
    currentPassword,
    setCurrentPassword,
    newPassword,
    setNewPassword,
    confirmPassword,
    setConfirmPassword,

    // handlers
    saveProfile,   // backend /members PATCH
    saveEmail,     // Firebase Auth でメール変更 & 変更メール送信
    savePassword,  // Firebase Auth でパスワード変更
  };
}
