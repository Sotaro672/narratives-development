// frontend/console/shell/src/auth/presentation/hook/useHeader.ts

import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";

import { useAuthContext } from "../../application/AuthContext";
import { useAuthActions } from "../../application/useAuthActions";

type UseHeaderParams = {
  username?: string;
  email?: string;
};

export function useHeader(_params: UseHeaderParams = {}) {
  const navigate = useNavigate();
  const [openAdmin, setOpenAdmin] = useState(false);
  const panelContainerRef = useRef<HTMLDivElement | null>(null);
  const triggerRef = useRef<HTMLButtonElement | null>(null);

  const { signOut } = useAuthActions();
  const { companyName, currentMember } = useAuthContext();

  useEffect(() => {
    const handleDocumentMouseDown = (event: MouseEvent) => {
      const target = event.target as Node;

      if (!panelContainerRef.current) {
        return;
      }

      if (panelContainerRef.current.contains(target)) {
        return;
      }

      setOpenAdmin(false);
    };

    document.addEventListener("mousedown", handleDocumentMouseDown);

    return () => {
      document.removeEventListener("mousedown", handleDocumentMouseDown);
    };
  }, []);

  useEffect(() => {
    const handleDocumentKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setOpenAdmin(false);
      }
    };

    document.addEventListener("keydown", handleDocumentKeyDown);

    return () => {
      document.removeEventListener("keydown", handleDocumentKeyDown);
    };
  }, []);

  const handleOpenCompanyDetail = () => {
    setOpenAdmin(false);
    navigate("/company");
  };

  const handleLogout = async () => {
    try {
      await signOut();
      setOpenAdmin(false);
    } catch (error: unknown) {
      console.error("logout failed", error);
    }
  };

  const handleToggleAdmin = () => {
    setOpenAdmin((current) => !current);
  };

  const brandMain = companyName ?? "Company Name";
  const fullName = currentMember?.displayName ?? "ゲスト";
  const displayEmail = currentMember?.email ?? "";

  return {
    openAdmin,
    panelContainerRef,
    triggerRef,
    brandMain,
    fullName,
    displayEmail,
    handleOpenCompanyDetail,
    handleToggleAdmin,
    handleLogout,
  };
}