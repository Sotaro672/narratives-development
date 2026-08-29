// frontend/amol/src/features/resale/presentation/models/resaleCreateProgress.ts

export type ResaleCreateProgressPhase =
  | "idle"
  | "preparing"
  | "uploading"
  | "saving"
  | "failed";

export type ResaleCreateProgress = {
  phase: ResaleCreateProgressPhase;
  title: string;
  message: string;
  percentage: number;
  transferredBytes: number;
  totalBytes: number;
  currentFileName: string;
  completedUploadCount: number;
  expectedUploadCount: number;
  errorMessage: string;
  isBrowserDependent: boolean;
  isBlockingNavigation: boolean;
};

export type ResaleCreateUploadProgressInput = {
  fileName?: string;
  transferredBytes: number;
  totalBytes: number;
  completedUploadCount: number;
  expectedUploadCount: number;
};

export type ResaleCreateSavingProgressInput = {
  transferredBytes?: number;
  totalBytes?: number;
  completedUploadCount?: number;
  expectedUploadCount?: number;
};

function normalizeBytes(value: number): number {
  if (!Number.isFinite(value)) {
    return 0;
  }

  return Math.max(0, value);
}

function normalizeCount(value: number): number {
  if (!Number.isFinite(value)) {
    return 0;
  }

  return Math.max(0, Math.floor(value));
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
  phase: ResaleCreateProgressPhase,
): ResaleCreateProgress {
  const isUploading = phase === "uploading";

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
        completedUploadCount: 0,
        expectedUploadCount: 0,
        errorMessage: "",
        isBrowserDependent: false,
        isBlockingNavigation: false,
      };

    case "preparing":
      return {
        phase,
        title: "出品準備中",
        message: "出品情報の登録準備をしています。",
        percentage: 0,
        transferredBytes: 0,
        totalBytes: 0,
        currentFileName: "",
        completedUploadCount: 0,
        expectedUploadCount: 0,
        errorMessage: "",
        isBrowserDependent: false,
        isBlockingNavigation: false,
      };

    case "uploading":
      return {
        phase,
        title: "商品状態画像を転送中",
        message: "画像転送が完了するまで、この画面を閉じたり別のページへ移動したりしないでください。",
        percentage: 0,
        transferredBytes: 0,
        totalBytes: 0,
        currentFileName: "",
        completedUploadCount: 0,
        expectedUploadCount: 0,
        errorMessage: "",
        isBrowserDependent: true,
        isBlockingNavigation: true,
      };

    case "saving":
      return {
        phase,
        title: "出品情報を保存中",
        message: "画像転送は完了しています。出品情報を保存しています。",
        percentage: 100,
        transferredBytes: 0,
        totalBytes: 0,
        currentFileName: "",
        completedUploadCount: 0,
        expectedUploadCount: 0,
        errorMessage: "",
        isBrowserDependent: false,
        isBlockingNavigation: false,
      };

    case "failed":
      return {
        phase,
        title: "出品に失敗しました",
        message: "出品処理中にエラーが発生しました。",
        percentage: 0,
        transferredBytes: 0,
        totalBytes: 0,
        currentFileName: "",
        completedUploadCount: 0,
        expectedUploadCount: 0,
        errorMessage: "",
        isBrowserDependent: false,
        isBlockingNavigation: false,
      };

    default:
      return {
        phase: "idle",
        title: "",
        message: "",
        percentage: 0,
        transferredBytes: 0,
        totalBytes: 0,
        currentFileName: "",
        completedUploadCount: 0,
        expectedUploadCount: 0,
        errorMessage: "",
        isBrowserDependent: isUploading,
        isBlockingNavigation: isUploading,
      };
  }
}

export function createInitialResaleCreateProgress(): ResaleCreateProgress {
  return createBaseProgress("idle");
}

export function createPreparingResaleCreateProgress(): ResaleCreateProgress {
  return createBaseProgress("preparing");
}

export function createUploadingResaleCreateProgress(
  input: ResaleCreateUploadProgressInput,
): ResaleCreateProgress {
  const transferredBytes = normalizeBytes(input.transferredBytes);
  const totalBytes = normalizeBytes(input.totalBytes);
  const completedUploadCount = normalizeCount(input.completedUploadCount);
  const expectedUploadCount = normalizeCount(input.expectedUploadCount);

  return {
    ...createBaseProgress("uploading"),
    percentage: calculatePercentage(
      transferredBytes,
      totalBytes,
    ),
    transferredBytes,
    totalBytes,
    currentFileName: input.fileName?.trim() ?? "",
    completedUploadCount,
    expectedUploadCount,
  };
}

export function createSavingResaleCreateProgress(
  input: ResaleCreateSavingProgressInput = {},
): ResaleCreateProgress {
  const transferredBytes = normalizeBytes(
    input.transferredBytes ?? 0,
  );
  const totalBytes = normalizeBytes(
    input.totalBytes ?? 0,
  );
  const completedUploadCount = normalizeCount(
    input.completedUploadCount ?? 0,
  );
  const expectedUploadCount = normalizeCount(
    input.expectedUploadCount ?? 0,
  );

  return {
    ...createBaseProgress("saving"),
    percentage: totalBytes > 0 ? 100 : 0,
    transferredBytes:
      totalBytes > 0
        ? totalBytes
        : transferredBytes,
    totalBytes,
    completedUploadCount:
      expectedUploadCount > 0
        ? expectedUploadCount
        : completedUploadCount,
    expectedUploadCount,
  };
}

export function createFailedResaleCreateProgress(
  errorMessage: string,
): ResaleCreateProgress {
  return {
    ...createBaseProgress("failed"),
    errorMessage: errorMessage.trim(),
  };
}

export function isResaleCreateProgressVisible(
  progress: ResaleCreateProgress,
): boolean {
  return progress.phase !== "idle";
}

export function shouldBlockResaleCreateNavigation(
  progress: ResaleCreateProgress,
): boolean {
  return progress.isBlockingNavigation;
}