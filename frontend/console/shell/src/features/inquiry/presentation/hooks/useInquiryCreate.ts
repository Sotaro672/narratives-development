// frontend/console/shell/src/features/inquiry/presentation/hooks/useInquiryCreate.ts

import {
  type ChangeEvent,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useNavigate } from "react-router-dom";

import { useAuthContext } from "../../../../auth/application/AuthContext";
import {
  MAX_INQUIRY_CREATE_IMAGES,
  MAX_INQUIRY_CREATE_IMAGE_SIZE_BYTES,
  MAX_INQUIRY_CREATE_IMAGE_SIZE_MB,
  MAX_INQUIRY_CREATE_MESSAGE_LENGTH,
} from "../../constants/inquiryCreate";
import {
  createInquiryContactHTTP,
  uploadInquiryContactAttachments,
} from "../../infrastructure/inquiryContactRepositoryHTTP";

export type InquiryCreateAttachment = {
  id: string;
  file: File;
  previewUrl: string;
};

function createAttachmentId(): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }

  return `${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

export function useInquiryCreate() {
  const navigate = useNavigate();
  const {
    user,
    currentMember,
    companyName,
    loading,
    loadingMember,
  } = useAuthContext();

  const [message, setMessage] = useState("");
  const [attachments, setAttachments] = useState<InquiryCreateAttachment[]>([]);
  const [submitting, setSubmitting] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(0);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const attachmentsRef = useRef<InquiryCreateAttachment[]>([]);
  attachmentsRef.current = attachments;

  useEffect(() => {
    return () => {
      attachmentsRef.current.forEach((attachment) => {
        URL.revokeObjectURL(attachment.previewUrl);
      });
    };
  }, []);

  const canSubmit = useMemo(() => {
    const normalizedMessage = message.trim();
    const email =
      currentMember?.email?.trim() ||
      user?.email?.trim() ||
      "";

    return (
      !loading &&
      !loadingMember &&
      !submitting &&
      Boolean(email) &&
      normalizedMessage.length > 0 &&
      normalizedMessage.length <= MAX_INQUIRY_CREATE_MESSAGE_LENGTH
    );
  }, [
    currentMember?.email,
    loading,
    loadingMember,
    message,
    submitting,
    user?.email,
  ]);

  const handleMessageChange = (value: string) => {
    setMessage(value);

    if (errorMessage) {
      setErrorMessage(null);
    }
  };

  const handleFilesSelected = (
    event: ChangeEvent<HTMLInputElement>,
  ) => {
    const selectedFiles = Array.from(event.target.files ?? []);
    event.target.value = "";

    if (selectedFiles.length === 0) {
      return;
    }

    const imageFiles = selectedFiles.filter((file) =>
      file.type.startsWith("image/"),
    );

    if (imageFiles.length !== selectedFiles.length) {
      window.alert("添付できるファイルは画像のみです。");
    }

    const validFiles = imageFiles.filter((file) => {
      if (file.size > MAX_INQUIRY_CREATE_IMAGE_SIZE_BYTES) {
        window.alert(
          `${file.name} は${MAX_INQUIRY_CREATE_IMAGE_SIZE_MB}MBを超えているため添付できません。`,
        );
        return false;
      }

      return true;
    });

    if (validFiles.length === 0) {
      return;
    }

    const remainingCount =
      MAX_INQUIRY_CREATE_IMAGES - attachments.length;

    if (remainingCount <= 0) {
      window.alert(
        `添付画像は最大${MAX_INQUIRY_CREATE_IMAGES}枚までです。`,
      );
      return;
    }

    const filesToAdd = validFiles.slice(0, remainingCount);

    if (validFiles.length > remainingCount) {
      window.alert(
        `添付画像は最大${MAX_INQUIRY_CREATE_IMAGES}枚までです。`,
      );
    }

    const nextAttachments = filesToAdd.map((file) => ({
      id: createAttachmentId(),
      file,
      previewUrl: URL.createObjectURL(file),
    }));

    setAttachments((current) => [
      ...current,
      ...nextAttachments,
    ]);
  };

  const handleRemoveAttachment = (id: string) => {
    if (submitting) {
      return;
    }

    setAttachments((current) => {
      const target = current.find(
        (attachment) => attachment.id === id,
      );

      if (target) {
        URL.revokeObjectURL(target.previewUrl);
      }

      return current.filter(
        (attachment) => attachment.id !== id,
      );
    });
  };

  const handleBack = () => {
    if (submitting) {
      return;
    }

    navigate("/inquiry");
  };

  const handleSubmit = async () => {
    if (!canSubmit) {
      return;
    }

    const trimmedMessage = message.trim();

    const email =
      currentMember?.email?.trim() ||
      user?.email?.trim() ||
      "";

    const name =
      currentMember?.displayName?.trim() ||
      user?.displayName?.trim() ||
      email;

    if (!name) {
      setErrorMessage(
        "問い合わせ送信者の名前を確認できませんでした。",
      );
      return;
    }

    if (!email) {
      setErrorMessage(
        "問い合わせ送信者のメールアドレスを確認できませんでした。",
      );
      return;
    }

    if (!trimmedMessage) {
      setErrorMessage("問い合わせ本文を入力してください。");
      return;
    }

    setSubmitting(true);
    setUploadProgress(0);
    setErrorMessage(null);

    try {
      const attachmentImageIds =
        await uploadInquiryContactAttachments(
          attachments.map((attachment) => attachment.file),
          ({ totalProgress }) => {
            setUploadProgress(totalProgress);
          },
        );

      await createInquiryContactHTTP({
        name,
        email,
        company: companyName?.trim() ?? "",
        message: trimmedMessage,
        attachmentImageIds,
      });

      attachments.forEach((attachment) => {
        URL.revokeObjectURL(attachment.previewUrl);
      });

      setAttachments([]);
      setMessage("");
      setUploadProgress(100);

      window.alert("AMOLへのお問い合わせを送信しました。");
      navigate("/inquiry");
    } catch (error) {
      setErrorMessage(
        error instanceof Error
          ? error.message
          : "お問い合わせの送信に失敗しました。",
      );
    } finally {
      setSubmitting(false);
    }
  };

  return {
    message,
    attachments,
    submitting,
    uploadProgress,
    errorMessage,
    canSubmit,
    maxMessageLength: MAX_INQUIRY_CREATE_MESSAGE_LENGTH,
    maxImages: MAX_INQUIRY_CREATE_IMAGES,
    maxImageSizeMB: MAX_INQUIRY_CREATE_IMAGE_SIZE_MB,
    handleMessageChange,
    handleFilesSelected,
    handleRemoveAttachment,
    handleBack,
    handleSubmit,
  };
}