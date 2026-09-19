// frontend/console/shell/src/features/brand/presentation/model/brandProgress.ts

export type BrandProgressVariant = "create" | "update";

export type BrandProgressPhase =
  | "idle"
  | "preparing"
  | "uploading"
  | "saving"
  | "completed"
  | "failed";

export type BrandProgress = {
  variant: BrandProgressVariant;
  phase: BrandProgressPhase;
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

export type BrandUploadProgressInput = {
  variant: BrandProgressVariant;
  fileName: string;
  transferredBytes: number;
  totalBytes: number;
  completedUploadCount: number;
  expectedUploadCount: number;
  title?: string;
  message?: string;
};

export type BrandSavingProgressInput = {
  variant: BrandProgressVariant;
  transferredBytes?: number;
  totalBytes?: number;
  completedUploadCount?: number;
  expectedUploadCount?: number;
  title?: string;
  message?: string;
};

export type BrandCompletedProgressInput = {
  variant: BrandProgressVariant;
  transferredBytes?: number;
  totalBytes?: number;
  completedUploadCount?: number;
  expectedUploadCount?: number;
  title?: string;
  message?: string;
};

export type BrandPreparingProgressInput = {
  variant: BrandProgressVariant;
  title?: string;
  message?: string;
};

export type BrandFailedProgressInput = {
  variant: BrandProgressVariant;
  title?: string;
  message?: string;
};

function clampPercentage(value: number): number {
  if (!Number.isFinite(value)) {
    return 0;
  }

  return Math.min(100, Math.max(0, Math.round(value)));
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
  variant: BrandProgressVariant,
  phase: BrandProgressPhase,
): string {
  switch (phase) {
    case "idle":
      return "";

    case "preparing":
      return variant === "create"
        ? "ブランドを登録中"
        : "ブランド情報を更新中";

    case "uploading":
      return "画像を転送中";

    case "saving":
      return "ブランド情報を保存中";

    case "completed":
      return variant === "create"
        ? "登録が完了しました"
        : "更新が完了しました";

    case "failed":
      return variant === "create"
        ? "登録に失敗しました"
        : "更新に失敗しました";
  }
}

function defaultMessage(
  variant: BrandProgressVariant,
  phase: BrandProgressPhase,
): string {
  switch (phase) {
    case "idle":
      return "";

    case "preparing":
      return variant === "create"
        ? "ブランド情報の登録準備をしています。"
        : "ブランド情報の更新準備をしています。";

    case "uploading":
      return "画像転送が完了するまで、この画面を閉じたり移動したりしないでください。";

    case "saving":
      return variant === "create"
        ? "画像転送が完了しました。ブランド情報を登録しています。"
        : "画像転送が完了しました。ブランド情報を更新しています。";

    case "completed":
      return variant === "create"
        ? "ブランドの登録が完了しました。"
        : "ブランド情報の更新が完了しました。";

    case "failed":
      return variant === "create"
        ? "ブランドの登録中にエラーが発生しました。"
        : "ブランド情報の更新中にエラーが発生しました。";
  }
}

function buildBaseProgress(
  variant: BrandProgressVariant,
  phase: BrandProgressPhase,
): BrandProgress {
  const browserDependent = phase === "uploading";

  return {
    variant,
    phase,
    title: defaultTitle(variant, phase),
    message: defaultMessage(variant, phase),
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

export function createInitialBrandProgress(
  variant: BrandProgressVariant,
): BrandProgress {
  return buildBaseProgress(variant, "idle");
}

export function createPreparingBrandProgress(
  input: BrandPreparingProgressInput,
): BrandProgress {
  return {
    ...buildBaseProgress(input.variant, "preparing"),
    title:
      input.title ??
      defaultTitle(input.variant, "preparing"),
    message:
      input.message ??
      defaultMessage(input.variant, "preparing"),
  };
}

export function createUploadingBrandProgress(
  input: BrandUploadProgressInput,
): BrandProgress {
  const transferredBytes =
    normalizeByteCount(input.transferredBytes);
  const totalBytes =
    normalizeByteCount(input.totalBytes);
  const completedUploadCount =
    normalizeCount(input.completedUploadCount);
  const expectedUploadCount =
    normalizeCount(input.expectedUploadCount);

  return {
    ...buildBaseProgress(input.variant, "uploading"),
    title:
      input.title ??
      defaultTitle(input.variant, "uploading"),
    message:
      input.message ??
      defaultMessage(input.variant, "uploading"),
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

export function createSavingBrandProgress(
  input: BrandSavingProgressInput,
): BrandProgress {
  const transferredBytes =
    normalizeByteCount(input.transferredBytes ?? 0);
  const totalBytes =
    normalizeByteCount(input.totalBytes ?? 0);
  const completedUploadCount =
    normalizeCount(input.completedUploadCount ?? 0);
  const expectedUploadCount =
    normalizeCount(input.expectedUploadCount ?? 0);

  return {
    ...buildBaseProgress(input.variant, "saving"),
    title:
      input.title ??
      defaultTitle(input.variant, "saving"),
    message:
      input.message ??
      defaultMessage(input.variant, "saving"),
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

export function createCompletedBrandProgress(
  input: BrandCompletedProgressInput,
): BrandProgress {
  const transferredBytes =
    normalizeByteCount(input.transferredBytes ?? 0);
  const totalBytes =
    normalizeByteCount(input.totalBytes ?? 0);
  const completedUploadCount =
    normalizeCount(input.completedUploadCount ?? 0);
  const expectedUploadCount =
    normalizeCount(input.expectedUploadCount ?? 0);

  return {
    ...buildBaseProgress(input.variant, "completed"),
    title:
      input.title ??
      defaultTitle(input.variant, "completed"),
    message:
      input.message ??
      defaultMessage(input.variant, "completed"),
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

export function createFailedBrandProgress(
  errorMessage: string,
  input: BrandFailedProgressInput,
): BrandProgress {
  return {
    ...buildBaseProgress(input.variant, "failed"),
    title:
      input.title ??
      defaultTitle(input.variant, "failed"),
    message:
      input.message ??
      defaultMessage(input.variant, "failed"),
    errorMessage:
      errorMessage || undefined,
  };
}

export function shouldBlockBrandNavigation(
  progress: BrandProgress,
): boolean {
  return progress.isBlockingNavigation;
}

export function isBrandProgressVisible(
  progress: BrandProgress,
): boolean {
  return progress.phase !== "idle";
}

export function isBrandProgressFinished(
  progress: BrandProgress,
): boolean {
  return progress.isTerminal;
}