// frontend/mall/src/features/avatar/models/avatarCreateProgress.ts

export type AvatarCreateProgressPhase =
  | "idle"
  | "preparing"
  | "uploading"
  | "saving"
  | "failed";

export type AvatarCreateProgress = {
  phase: AvatarCreateProgressPhase;
  title: string;
  message: string;
  percentage: number;
  transferredBytes: number;
  totalBytes: number;
  currentFileName: string;
  errorMessage: string;
  isBrowserDependent: boolean;
  isBlockingNavigation: boolean;
};

export type AvatarCreateUploadProgressInput = {
  fileName?: string;
  transferredBytes: number;
  totalBytes: number;
};

export type AvatarCreateSavingProgressInput = {
  transferredBytes?: number;
  totalBytes?: number;
};

function normalizeBytes(value: number): number {
  if (!Number.isFinite(value)) {
    return 0;
  }

  return Math.max(0, value);
}

function calculatePercentage(
  transferredBytes: number,
  totalBytes: number,
): number {
  if (totalBytes <= 0) {
    return 0;
  }

  return Math.min(
    100,
    Math.max(
      0,
      Math.round(
        (transferredBytes / totalBytes) * 100,
      ),
    ),
  );
}

function createBaseProgress(
  phase: AvatarCreateProgressPhase,
): AvatarCreateProgress {
  switch (phase) {
    case "idle":
      return {
        phase,
        title: "",
        message: "",
        percentage: 0,
        transferredBytes: 0,
        totalBytes: 0,
        currentFileName: "",
        errorMessage: "",
        isBrowserDependent: false,
        isBlockingNavigation: false,
      };

    case "preparing":
      return {
        phase,
        title: "アバターを保存する準備をしています",
        message: "アバター情報の保存準備をしています。",
        percentage: 0,
        transferredBytes: 0,
        totalBytes: 0,
        currentFileName: "",
        errorMessage: "",
        isBrowserDependent: false,
        isBlockingNavigation: false,
      };

    case "uploading":
      return {
        phase,
        title: "アバターアイコンを転送中",
        message:
          "画像転送が完了するまで、この画面を閉じたり別のページへ移動したりしないでください。",
        percentage: 0,
        transferredBytes: 0,
        totalBytes: 0,
        currentFileName: "",
        errorMessage: "",
        isBrowserDependent: true,
        isBlockingNavigation: true,
      };

    case "saving":
      return {
        phase,
        title: "アバター情報を保存中",
        message: "画像転送は完了しています。アバター情報を保存しています。",
        percentage: 100,
        transferredBytes: 0,
        totalBytes: 0,
        currentFileName: "",
        errorMessage: "",
        isBrowserDependent: false,
        isBlockingNavigation: false,
      };

    case "failed":
      return {
        phase,
        title: "アバターの保存に失敗しました",
        message: "アバターの保存処理中にエラーが発生しました。",
        percentage: 0,
        transferredBytes: 0,
        totalBytes: 0,
        currentFileName: "",
        errorMessage: "",
        isBrowserDependent: false,
        isBlockingNavigation: false,
      };
  }
}

export function createInitialAvatarCreateProgress(): AvatarCreateProgress {
  return createBaseProgress("idle");
}

export function createPreparingAvatarCreateProgress(): AvatarCreateProgress {
  return createBaseProgress("preparing");
}

export function createUploadingAvatarCreateProgress(
  input: AvatarCreateUploadProgressInput,
): AvatarCreateProgress {
  const transferredBytes = normalizeBytes(input.transferredBytes);
  const totalBytes = normalizeBytes(input.totalBytes);

  return {
    ...createBaseProgress("uploading"),
    percentage: calculatePercentage(
      transferredBytes,
      totalBytes,
    ),
    transferredBytes,
    totalBytes,
    currentFileName: input.fileName?.trim() ?? "",
  };
}

export function createSavingAvatarCreateProgress(
  input: AvatarCreateSavingProgressInput = {},
): AvatarCreateProgress {
  const transferredBytes = normalizeBytes(
    input.transferredBytes ?? 0,
  );
  const totalBytes = normalizeBytes(
    input.totalBytes ?? 0,
  );

  return {
    ...createBaseProgress("saving"),
    percentage: totalBytes > 0 ? 100 : 0,
    transferredBytes:
      totalBytes > 0
        ? totalBytes
        : transferredBytes,
    totalBytes,
  };
}

export function createFailedAvatarCreateProgress(
  errorMessage: string,
): AvatarCreateProgress {
  return {
    ...createBaseProgress("failed"),
    errorMessage: errorMessage.trim(),
  };
}

export function isAvatarCreateProgressVisible(
  progress: AvatarCreateProgress,
): boolean {
  return progress.phase !== "idle";
}

export function shouldBlockAvatarCreateNavigation(
  progress: AvatarCreateProgress,
): boolean {
  return progress.isBlockingNavigation;
}