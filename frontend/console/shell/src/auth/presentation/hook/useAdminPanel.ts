// frontend/console/shell/src/auth/presentation/hook/useAdminPanel.ts
import { useEffect, useState, useCallback } from "react";
import { auth } from "../../infrastructure/config/firebaseClient";
import {
  fetchCurrentMember,
  updateCurrentMemberProfile,
} from "../../application/memberService";

export function useAdminPanel() {
  // dialog flags
  const [showProfileDialog, setShowProfileDialog] = useState(false);
  const [showEmailDialog, setShowEmailDialog] = useState(false);
  const [showPasswordDialog, setShowPasswordDialog] = useState(false);

  // ★ 追加: currentMember の id
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
  // ★ プロフィール保存処理（Backend 経由で Firebase/DB 更新）
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
      // 成功したらダイアログを閉じる
      setShowProfileDialog(false);
    } catch (e) {
      console.error("[useAdminPanel] failed to update profile:", e);
      // TODO: 必要ならエラー用 state を追加して UI に表示
    }
  }, [memberId, firstName, lastName, firstNameKana, lastNameKana]);

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
    saveProfile,
  };
}
