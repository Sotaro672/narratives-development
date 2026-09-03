// frontend/console/shell/src/features/brand/presentation/model/brandCreateProgress.ts

export type BrandCreateProgressPhase =
  | "idle"
  | "creating"
  | "uploading"
  | "saving"
  | "completed"
  | "failed";

export type BrandCreateProgress = {
  phase: BrandCreateProgressPhase;
  title: string;
  message: string;
  percentage: number;
  transferredBytes: number;
  totalBytes: number;
  currentFileName?: string;
  completedUploadCount: number;
  expectedUploadCount: number;
  errorMessage?: string;
  isBrowserDependent: boolean;
  isBlockingNavigation: boolean;
  isTerminal: boolean;
};

export type BrandCreateUploadProgressInput = {
  fileName: string;
  transferredBytes: number;
  totalBytes: number;
  completedUploadCount: number;
  expectedUploadCount: number;
  title?: string;
  message?: string;
};

export type BrandCreateSavingProgressInput = {
  transferredBytes?: number;
  totalBytes?: number;
  completedUploadCount?: number;
  expectedUploadCount?: number;
  title?: string;
  message?: string;
};

export type BrandCreateCompletedProgressInput = {
  transferredBytes?: number;
  totalBytes?: number;
  completedUploadCount?: number;
  expectedUploadCount?: number;
  title?: string;
  message?: string;
};

function clampPercentage(value: number): number {
  if (!Number.isFinite(value)) {
    return 0;
  }

  return Math.min(
    100,
    Math.max(0, Math.round(value)),
  );
}

function normalizeByteCount(value: number): number {
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

  return clampPercentage(
    (transferredBytes / totalBytes) * 100,
  );
}

function defaultTitle(
  phase: BrandCreateProgressPhase,
): string {
  switch (phase) {
    case "idle":
      return "";

    case "creating":
      return "ブランドを登録中";

    case "uploading":
      return "画像を転送中";

    case "saving":
      return "ブランド情報を保存中";

    case "completed":
      return "登録が完了しました";

    case "failed":
      return "登録に失敗しました";
  }
}

function defaultMessage(
  phase: BrandCreateProgressPhase,
): string {
  switch (phase) {
    case "idle":
      return "";

    case "creating":
      return "ブランド情報を登録しています。";

    case "uploading":
      return "画像転送が完了するまで、この画面を閉じたり移動したりしないでください。";

    case "saving":
      return "画像転送が完了しました。ブランド情報を更新しています。";

    case "completed":
      return "ブランドの登録が完了しました。";

    case "failed":
      return "ブランドの登録中にエラーが発生しました。";
  }
}

function buildBaseProgress(
  phase: BrandCreateProgressPhase,
): BrandCreateProgress {
  const browserDependent = phase === "uploading";

  return {
    phase,
    title: defaultTitle(phase),
    message: defaultMessage(phase),
    percentage: 0,
    transferredBytes: 0,
    totalBytes: 0,
    completedUploadCount: 0,
    expectedUploadCount: 0,
    isBrowserDependent: browserDependent,
    isBlockingNavigation: browserDependent,
    isTerminal:
      phase === "completed" ||
      phase === "failed",
  };
}

export function createInitialBrandCreateProgress(): BrandCreateProgress {
  return buildBaseProgress("idle");
}

export function createCreatingBrandCreateProgress(
  input?: {
    title?: string;
    message?: string;
  },
): BrandCreateProgress {
  return {
    ...buildBaseProgress("creating"),
    title:
      input?.title ??
      defaultTitle("creating"),
    message:
      input?.message ??
      defaultMessage("creating"),
  };
}

export function createUploadingBrandCreateProgress(
  input: BrandCreateUploadProgressInput,
): BrandCreateProgress {
  const transferredBytes = normalizeByteCount(
    input.transferredBytes,
  );

  const totalBytes = normalizeByteCount(
    input.totalBytes,
  );

  const completedUploadCount = normalizeCount(
    input.completedUploadCount,
  );

  const expectedUploadCount = normalizeCount(
    input.expectedUploadCount,
  );

  return {
    ...buildBaseProgress("uploading"),
    title:
      input.title ??
      defaultTitle("uploading"),
    message:
      input.message ??
      defaultMessage("uploading"),
    percentage: calculatePercentage(
      transferredBytes,
      totalBytes,
    ),
    transferredBytes,
    totalBytes,
    currentFileName: input.fileName,
    completedUploadCount,
    expectedUploadCount,
  };
}

export function createSavingBrandCreateProgress(
  input?: BrandCreateSavingProgressInput,
): BrandCreateProgress {
  const transferredBytes = normalizeByteCount(
    input?.transferredBytes ?? 0,
  );

  const totalBytes = normalizeByteCount(
    input?.totalBytes ?? 0,
  );

  const completedUploadCount = normalizeCount(
    input?.completedUploadCount ?? 0,
  );

  const expectedUploadCount = normalizeCount(
    input?.expectedUploadCount ?? 0,
  );

  return {
    ...buildBaseProgress("saving"),
    title:
      input?.title ??
      defaultTitle("saving"),
    message:
      input?.message ??
      defaultMessage("saving"),
    percentage:
      totalBytes > 0
        ? calculatePercentage(
            transferredBytes,
            totalBytes,
          )
        : 0,
    transferredBytes,
    totalBytes,
    completedUploadCount,
    expectedUploadCount,
  };
}

export function createCompletedBrandCreateProgress(
  input?: BrandCreateCompletedProgressInput,
): BrandCreateProgress {
  const transferredBytes = normalizeByteCount(
    input?.transferredBytes ?? 0,
  );

  const totalBytes = normalizeByteCount(
    input?.totalBytes ?? 0,
  );

  const completedUploadCount = normalizeCount(
    input?.completedUploadCount ?? 0,
  );

  const expectedUploadCount = normalizeCount(
    input?.expectedUploadCount ?? 0,
  );

  return {
    ...buildBaseProgress("completed"),
    title:
      input?.title ??
      defaultTitle("completed"),
    message:
      input?.message ??
      defaultMessage("completed"),
    percentage:
      totalBytes > 0
        ? 100
        : 0,
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

export function createFailedBrandCreateProgress(
  errorMessage: string,
  input?: {
    title?: string;
    message?: string;
  },
): BrandCreateProgress {
  return {
    ...buildBaseProgress("failed"),
    title:
      input?.title ??
      defaultTitle("failed"),
    message:
      input?.message ??
      defaultMessage("failed"),
    errorMessage:
      errorMessage || undefined,
  };
}

export function shouldBlockBrandCreateNavigation(
  progress: BrandCreateProgress,
): boolean {
  return progress.phase === "uploading";
}

export function isBrandCreateProgressVisible(
  progress: BrandCreateProgress,
): boolean {
  return progress.phase !== "idle";
}

export function isBrandCreateProgressFinished(
  progress: BrandCreateProgress,
): boolean {
  return progress.isTerminal;
}