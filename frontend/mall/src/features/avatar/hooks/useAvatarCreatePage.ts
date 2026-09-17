// frontend/mall/src/features/avatar/hooks/useAvatarCreatePage.ts

import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { signOut as firebaseSignOut } from "firebase/auth";

import { auth } from "../../../lib/firebase";
import {
  createFailedAvatarCreateProgress,
  createInitialAvatarCreateProgress,
  createPreparingAvatarCreateProgress,
  createSavingAvatarCreateProgress,
  createUploadingAvatarCreateProgress,
} from "../models/avatarCreateProgress";
import { AvatarCreateService } from "../services/avatarCreateService";
import {
  cropAvatarImage,
  type AvatarCropPosition,
} from "../utils/cropAvatarImage";
import type { AvatarFormMode } from "../../shared/types/avatar";

const INITIAL_ICON_POSITION: AvatarCropPosition = {
  x: 0,
  y: 0,
};

const INITIAL_ICON_SCALE = 1;

function revokePreviewUrl(url: string | null) {
  if (url?.startsWith("blob:")) {
    URL.revokeObjectURL(url);
  }
}

export function useAvatarCreatePage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const from = searchParams.get("from");

  const [mode, setMode] = useState<AvatarFormMode>("create");
  const [avatarId, setAvatarId] = useState("");
  const [avatarName, setAvatarName] = useState("");
  const [profile, setProfile] = useState("");
  const [externalLink, setExternalLink] = useState("");

  const [iconFile, setIconFile] = useState<File | null>(null);
  const [iconPreviewUrl, setIconPreviewUrl] = useState<string | null>(null);
  const [iconFileName, setIconFileName] = useState<string | null>(null);
  const [iconMimeType, setIconMimeType] = useState<string | null>(null);
  const [iconPosition, setIconPosition] = useState<AvatarCropPosition>(
    INITIAL_ICON_POSITION,
  );
  const [iconScale, setIconScale] = useState(INITIAL_ICON_SCALE);
  const [iconViewportSize, setIconViewportSize] = useState(0);

  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState("");
  const [createdAvatarId, setCreatedAvatarId] = useState("");
  const [successRedirectTo, setSuccessRedirectTo] = useState("");

  const [progress, setProgress] = useState(
    createInitialAvatarCreateProgress,
  );
  const [progressOpen, setProgressOpen] = useState(false);

  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const service = useMemo(() => new AvatarCreateService({ auth }), []);

  const loggedIn = auth.currentUser !== null;
  const backTo = useMemo(() => service.backTo(from), [from, service]);
  const canSave = useMemo(
    () => !loading && !saving && avatarName.length > 0,
    [avatarName, loading, saving],
  );
  const isSuccessMessage = useMemo(
    () => msg.includes("作成しました") || msg.includes("保存しました"),
    [msg],
  );

  const pageTitle = mode === "edit" ? "アバター編集" : "アバター作成";
  const saveButtonLabel =
    saving ? "保存中..." : mode === "edit" ? "保存する" : "作成する";

  function resetIconCrop() {
    setIconPosition(INITIAL_ICON_POSITION);
    setIconScale(INITIAL_ICON_SCALE);
    setIconViewportSize(0);
  }

  useEffect(() => {
    let cancelled = false;

    async function loadAvatar() {
      if (!auth.currentUser) {
        setLoading(false);
        return;
      }

      setLoading(true);
      setMsg("");

      try {
        const currentAvatar = await service.fetchMine();
        if (cancelled) return;

        if (!currentAvatar) {
          setMode("create");
          setAvatarId("");
          setAvatarName("");
          setProfile("");
          setExternalLink("");
          setIconFile(null);
          setIconPreviewUrl(null);
          setIconFileName(null);
          setIconMimeType(null);
          resetIconCrop();
          return;
        }

        setMode("edit");
        setAvatarId(currentAvatar.avatarId);
        setAvatarName(currentAvatar.avatarName);
        setProfile(currentAvatar.profile ?? "");
        setExternalLink(currentAvatar.externalLink ?? "");
        setIconFile(null);
        setIconPreviewUrl(currentAvatar.avatarIcon ?? null);
        setIconFileName(null);
        setIconMimeType(null);
        resetIconCrop();
      } catch (error) {
        if (cancelled) return;

        setMode("create");
        setMsg(error instanceof Error ? error.message : String(error));
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void loadAvatar();

    return () => {
      cancelled = true;
    };
  }, [service]);

  useEffect(() => {
    return () => {
      revokePreviewUrl(iconPreviewUrl);
    };
  }, [iconPreviewUrl]);

  function clearMessage() {
    if (msg) setMsg("");
  }

  function openIconPicker() {
    fileInputRef.current?.click();
  }

  function pickIcon(file: File | null) {
    setMsg("");
    revokePreviewUrl(iconPreviewUrl);

    const result = service.pickIconWeb(file);

    if (!result) {
      setMsg("画像選択をキャンセルしました。");
      return;
    }

    if (result.error) {
      setIconFile(null);
      setIconPreviewUrl(null);
      setIconFileName(null);
      setIconMimeType(null);
      resetIconCrop();
      setMsg(result.error);
      return;
    }

    setIconFile(result.file);
    setIconPreviewUrl(result.previewUrl);
    setIconFileName(result.fileName);
    setIconMimeType(result.mimeType);
    resetIconCrop();
    setMsg("アイコン画像を選択しました。");
  }

  function clearIcon() {
    revokePreviewUrl(iconPreviewUrl);
    setIconFile(null);
    setIconPreviewUrl(null);
    setIconFileName(null);
    setIconMimeType(null);
    resetIconCrop();
  }

  function handleIconPreviewError() {
    if (!iconFile) {
      setIconPreviewUrl(null);
    }
  }

  function handleCloseProgress() {
    if (progress.isBlockingNavigation) {
      return;
    }

    setProgressOpen(false);
    setProgress(createInitialAvatarCreateProgress());
  }

  async function signOut() {
    setMsg("");

    try {
      await firebaseSignOut(auth);
      navigate("/", { replace: true });
    } catch (error) {
      setMsg(
        error instanceof Error
          ? error.message
          : "サインアウトに失敗しました。",
      );
    }
  }

  async function save() {
    if (saving) return false;

    setSaving(true);
    setMsg("");
    setCreatedAvatarId("");
    setSuccessRedirectTo("");

    if (iconFile) {
      setProgress(createPreparingAvatarCreateProgress());
      setProgressOpen(true);
    } else {
      setProgress(createInitialAvatarCreateProgress());
      setProgressOpen(false);
    }

    try {
      let uploadIconFile: File | null = null;

      if (iconFile) {
        if (iconViewportSize <= 0) {
          throw new Error(
            "アイコン画像の切り抜き領域を取得できませんでした。画像を選択し直してください。",
          );
        }

        uploadIconFile = await cropAvatarImage({
          file: iconFile,
          position: iconPosition,
          scale: iconScale,
          viewportSize: iconViewportSize,
        });
      }

      const handleUploadProgress = ({
        transferredBytes,
        totalBytes,
      }: {
        transferredBytes: number;
        totalBytes: number;
        percentage: number;
      }) => {
        if (!uploadIconFile) {
          return;
        }

        setProgress(
          createUploadingAvatarCreateProgress({
            fileName: uploadIconFile.name,
            transferredBytes,
            totalBytes,
          }),
        );
      };

      const handleUploadCompleted = () => {
        if (!uploadIconFile) {
          return;
        }

        setProgress(
          createSavingAvatarCreateProgress({
            transferredBytes: uploadIconFile.size,
            totalBytes: uploadIconFile.size,
          }),
        );
      };

      let savedAvatarId: string;

      if (mode === "edit" && avatarId) {
        const result = await service.update({
          avatarId,
          avatarNameRaw: avatarName,
          profileRaw: profile,
          externalLinkRaw: externalLink,
          iconFile: uploadIconFile,
          onUploadProgress: handleUploadProgress,
          onUploadCompleted: handleUploadCompleted,
        });

        setMsg(result.message);

        if (!result.ok) {
          if (uploadIconFile) {
            setProgress(
              createFailedAvatarCreateProgress(result.message),
            );
          }

          return false;
        }

        savedAvatarId = result.avatarId;
      } else {
        const result = await service.save({
          avatarNameRaw: avatarName,
          profileRaw: profile,
          externalLinkRaw: externalLink,
          iconFile: uploadIconFile,
          onUploadProgress: handleUploadProgress,
          onUploadCompleted: handleUploadCompleted,
        });

        setMsg(result.message);

        if (!result.ok) {
          if (uploadIconFile) {
            setProgress(
              createFailedAvatarCreateProgress(result.message),
            );
          }

          return false;
        }

        savedAvatarId = result.createdAvatarId;
      }

      setCreatedAvatarId(savedAvatarId);
      setSuccessRedirectTo(backTo);
      navigate(backTo, { replace: true });

      return true;
    } catch (error) {
      const errorMessage =
        error instanceof Error
          ? error.message
          : String(error);

      setMsg(errorMessage);

      if (iconFile) {
        setProgress(
          createFailedAvatarCreateProgress(errorMessage),
        );
      }

      return false;
    } finally {
      setSaving(false);
    }
  }

  return {
    mode,
    avatarId,
    avatarName,
    setAvatarName,
    profile,
    setProfile,
    externalLink,
    setExternalLink,
    iconFile,
    iconPreviewUrl,
    iconFileName,
    iconMimeType,
    iconPosition,
    setIconPosition,
    iconScale,
    setIconScale,
    iconViewportSize,
    setIconViewportSize,
    fileInputRef,
    loading,
    saving,
    msg,
    createdAvatarId,
    successRedirectTo,
    loggedIn,
    backTo,
    canSave,
    isSuccessMessage,
    pageTitle,
    saveButtonLabel,
    progress,
    progressOpen,
    handleCloseProgress,
    clearMessage,
    openIconPicker,
    pickIcon,
    clearIcon,
    handleIconPreviewError,
    signOut,
    save,
  };
}