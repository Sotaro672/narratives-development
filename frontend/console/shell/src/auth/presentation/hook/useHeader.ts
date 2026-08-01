// frontend/console/shell/src/auth/presentation/hook/useHeader.ts

import {
  useEffect,
  useRef,
  useState,
} from "react";

import {
  useAuthContext,
} from "../../application/AuthContext";
import {
  useAuthActions,
} from "../../application/useAuthActions";

type UseHeaderParams = {
  username?: string;
  email?: string;
};

export function useHeader({
  username = "管理者",
  email = "admin@narratives.com",
}: UseHeaderParams) {
  const [
    openAdmin,
    setOpenAdmin,
  ] = useState(false);

  const panelContainerRef =
    useRef<HTMLDivElement | null>(null);

  const triggerRef =
    useRef<HTMLButtonElement | null>(null);

  const {
    signOut,
  } = useAuthActions();

  const {
    user,
    companyName,
    currentMember,
  } = useAuthContext();

  // ─────────────────────────────────────────────
  // 外側クリックで閉じる
  // ─────────────────────────────────────────────
  useEffect(() => {
    const handleDocumentMouseDown = (
      event: MouseEvent,
    ) => {
      const target = event.target as Node;

      if (!panelContainerRef.current) {
        return;
      }

      if (
        panelContainerRef.current.contains(
          target,
        )
      ) {
        return;
      }

      setOpenAdmin(false);
    };

    document.addEventListener(
      "mousedown",
      handleDocumentMouseDown,
    );

    return () => {
      document.removeEventListener(
        "mousedown",
        handleDocumentMouseDown,
      );
    };
  }, []);

  // ─────────────────────────────────────────────
  // Escキーで閉じる
  // ─────────────────────────────────────────────
  useEffect(() => {
    const handleDocumentKeyDown = (
      event: KeyboardEvent,
    ) => {
      if (event.key === "Escape") {
        setOpenAdmin(false);
      }
    };

    document.addEventListener(
      "keydown",
      handleDocumentKeyDown,
    );

    return () => {
      document.removeEventListener(
        "keydown",
        handleDocumentKeyDown,
      );
    };
  }, []);

  // ─────────────────────────────────────────────
  // ログアウト
  // ─────────────────────────────────────────────
  const handleLogout = async () => {
    try {
      await signOut();
      setOpenAdmin(false);
    } catch (error: unknown) {
      console.error(
        "logout failed",
        error,
      );
    }
  };

  const handleToggleAdmin = () => {
    setOpenAdmin(
      (current) => !current,
    );
  };

  // 会社名
  const brandMain =
    companyName?.trim() ||
    "Company Name";

  // 表示名
  const fullName =
    currentMember?.displayName.trim() ||
    `${currentMember?.lastName ?? ""} ${
      currentMember?.firstName ?? ""
    }`.trim() ||
    user?.displayName?.trim() ||
    user?.email?.trim() ||
    username ||
    "ゲスト";

  // メールアドレス
  const displayEmail =
    currentMember?.email.trim() ||
    user?.email?.trim() ||
    email;

  return {
    openAdmin,
    panelContainerRef,
    triggerRef,

    brandMain,
    fullName,
    displayEmail,

    handleToggleAdmin,
    handleLogout,
  };
}