// frontend/console/shell/src/features/tokenBlueprint/presentation/model/tokenBlueprintProgress.ts

import type {
  TokenBlueprintCreateOperation,
  TokenBlueprintCreateOperationStatus,
} from "../../infrastructure/repository/tokenBlueprintCreateOperationRepositoryHTTP";

export type TokenBlueprintProgressPhase =
  | "idle"
  | "preparing"
  | "uploading"
  | "saving"
  | "completed"
  | "failed";

export type TokenBlueprintCreateProgressPhase =
  | "idle"
  | "starting"
  | "uploading"
  | "queued"
  | "processing"
  | "completed"
  | "failed_retryable"
  | "failed_fatal";

export type TokenBlueprintProgressTarget = "icon" | "content";
export type TokenBlueprintCreateUploadTarget = TokenBlueprintProgressTarget;

export type TokenBlueprintProgress = {
  phase: TokenBlueprintProgressPhase;
  title: string;
  message: string;
  percentage: number;
  transferredBytes: number;
  totalBytes: number;
  currentFileName?: string;
  currentUploadTarget?: TokenBlueprintProgressTarget;
  completedUploadCount: number;
  expectedUploadCount: number;
  errorMessage?: string;
  isBrowserDependent: boolean;
  isBlockingNavigation: boolean;
  isTerminal: boolean;
};

export type TokenBlueprintCreateProgress = Omit<
  TokenBlueprintProgress,
  "phase"
> & {
  phase: TokenBlueprintCreateProgressPhase;
  operationId?: string;
  tokenBlueprintId?: string;
  retryCount: number;
  maxRetries: number;
  canRetry: boolean;
};

export type TokenBlueprintUploadProgressInput = {
  target: TokenBlueprintProgressTarget;
  fileName: string;
  transferredBytes: number;
  totalBytes: number;
  completedUploadCount: number;
  expectedUploadCount: number;
  title?: string;
  message?: string;
};

export type TokenBlueprintCreateUploadProgressInput = {
  operation?: TokenBlueprintCreateOperation | null;
  target: TokenBlueprintCreateUploadTarget;
  fileName: string;
  transferredBytes: number;
  totalBytes: number;
  completedUploadCount: number;
  expectedUploadCount: number;
};

export type TokenBlueprintSavingProgressInput = {
  completedUploadCount?: number;
  expectedUploadCount?: number;
  transferredBytes?: number;
  totalBytes?: number;
  title?: string;
  message?: string;
};

export type TokenBlueprintCompletedProgressInput = {
  completedUploadCount?: number;
  expectedUploadCount?: number;
  transferredBytes?: number;
  totalBytes?: number;
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
  phase: TokenBlueprintProgressPhase,
): string {
  switch (phase) {
    case "idle":
      return "";
    case "preparing":
      return "保存準備中";
    case "uploading":
      return "ファイルを転送中";
    case "saving":
      return "トークン設計を保存中";
    case "completed":
      return "保存が完了しました";
    case "failed":
      return "保存に失敗しました";
  }
}

function defaultMessage(
  phase: TokenBlueprintProgressPhase,
): string {
  switch (phase) {
    case "idle":
      return "";
    case "preparing":
      return "トークン設計の保存準備をしています。";
    case "uploading":
      return "ファイル転送が完了するまで、この画面を閉じたり移動したりしないでください。";
    case "saving":
      return "変更内容を保存しています。この画面を閉じたり移動したりしないでください。";
    case "completed":
      return "トークン設計の保存が完了しました。";
    case "failed":
      return "トークン設計の保存中にエラーが発生しました。";
  }
}

function buildBaseProgress(
  phase: TokenBlueprintProgressPhase,
): TokenBlueprintProgress {
  const browserDependent =
    phase === "preparing" ||
    phase === "uploading" ||
    phase === "saving";

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

export function createInitialTokenBlueprintProgress(): TokenBlueprintProgress {
  return buildBaseProgress("idle");
}

export function createPreparingTokenBlueprintProgress(
  input?: {
    title?: string;
    message?: string;
  },
): TokenBlueprintProgress {
  return {
    ...buildBaseProgress("preparing"),
    title:
      input?.title ??
      defaultTitle("preparing"),
    message:
      input?.message ??
      defaultMessage("preparing"),
  };
}

export function createUploadingTokenBlueprintProgress(
  input: TokenBlueprintUploadProgressInput,
): TokenBlueprintProgress {
  const transferredBytes =
    normalizeByteCount(input.transferredBytes);
  const totalBytes =
    normalizeByteCount(input.totalBytes);
  const completedUploadCount =
    normalizeCount(input.completedUploadCount);
  const expectedUploadCount =
    normalizeCount(input.expectedUploadCount);

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
    currentUploadTarget: input.target,
    completedUploadCount,
    expectedUploadCount,
  };
}

export function createSavingTokenBlueprintProgress(
  input?: TokenBlueprintSavingProgressInput,
): TokenBlueprintProgress {
  const transferredBytes =
    normalizeByteCount(
      input?.transferredBytes ?? 0,
    );
  const totalBytes =
    normalizeByteCount(
      input?.totalBytes ?? 0,
    );
  const completedUploadCount =
    normalizeCount(
      input?.completedUploadCount ?? 0,
    );
  const expectedUploadCount =
    normalizeCount(
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

export function createCompletedTokenBlueprintProgress(
  input?: TokenBlueprintCompletedProgressInput,
): TokenBlueprintProgress {
  const transferredBytes =
    normalizeByteCount(
      input?.transferredBytes ?? 0,
    );
  const totalBytes =
    normalizeByteCount(
      input?.totalBytes ?? 0,
    );
  const completedUploadCount =
    normalizeCount(
      input?.completedUploadCount ?? 0,
    );
  const expectedUploadCount =
    normalizeCount(
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

export function createFailedTokenBlueprintProgress(
  errorMessage: string,
  input?: {
    title?: string;
    message?: string;
  },
): TokenBlueprintProgress {
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

export function shouldBlockTokenBlueprintNavigation(
  progress: TokenBlueprintProgress,
): boolean {
  return progress.isBlockingNavigation;
}

export function isTokenBlueprintProgressVisible(
  progress: TokenBlueprintProgress,
): boolean {
  return progress.phase !== "idle";
}

export function isTokenBlueprintProgressFinished(
  progress: TokenBlueprintProgress,
): boolean {
  return progress.isTerminal;
}

function operationStatusToCreatePhase(
  status: TokenBlueprintCreateOperationStatus,
): TokenBlueprintCreateProgressPhase {
  switch (status) {
    case "waiting_upload":
      return "uploading";
    case "queued":
      return "queued";
    case "processing":
      return "processing";
    case "completed":
      return "completed";
    case "failed_retryable":
      return "failed_retryable";
    case "failed_fatal":
      return "failed_fatal";
  }
}

function createTokenBlueprintCreateProgress(
  base: TokenBlueprintProgress,
  phase: TokenBlueprintCreateProgressPhase,
  overrides?: Partial<
    Omit<TokenBlueprintCreateProgress, "phase">
  >,
): TokenBlueprintCreateProgress {
  return {
    ...base,
    ...overrides,
    phase,
    retryCount: overrides?.retryCount ?? 0,
    maxRetries: overrides?.maxRetries ?? 0,
    canRetry: overrides?.canRetry ?? false,
  };
}

export function createInitialTokenBlueprintCreateProgress(): TokenBlueprintCreateProgress {
  return createTokenBlueprintCreateProgress(
    createInitialTokenBlueprintProgress(),
    "idle",
  );
}

export function createStartingTokenBlueprintCreateProgress(): TokenBlueprintCreateProgress {
  return createTokenBlueprintCreateProgress(
    createInitialTokenBlueprintProgress(),
    "starting",
    {
      title: "作成準備中",
      message: "トークン設計の作成準備をしています。",
      isBrowserDependent: false,
      isBlockingNavigation: false,
      isTerminal: false,
    },
  );
}

export function createUploadingTokenBlueprintCreateProgress(
  input: TokenBlueprintCreateUploadProgressInput,
): TokenBlueprintCreateProgress {
  const base = createUploadingTokenBlueprintProgress({
    target: input.target,
    fileName: input.fileName,
    transferredBytes: input.transferredBytes,
    totalBytes: input.totalBytes,
    completedUploadCount: input.completedUploadCount,
    expectedUploadCount: input.expectedUploadCount,
  });

  return createTokenBlueprintCreateProgress(
    base,
    "uploading",
    {
      operationId: input.operation?.id,
      tokenBlueprintId:
        input.operation?.tokenBlueprintId,
      retryCount:
        input.operation?.retryCount ?? 0,
      maxRetries:
        input.operation?.maxRetries ?? 0,
    },
  );
}

export function createTokenBlueprintCreateProgressFromOperation(
  operation: TokenBlueprintCreateOperation,
): TokenBlueprintCreateProgress {
  const phase =
    operationStatusToCreatePhase(operation.status);

  const completedUploadCount =
    normalizeCount(operation.completedUploadCount);
  const expectedUploadCount =
    normalizeCount(operation.expectedUploadCount);

  const uploadPercentage =
    expectedUploadCount > 0
      ? clampPercentage(
          (completedUploadCount /
            expectedUploadCount) *
            100,
        )
      : phase === "queued" ||
          phase === "processing" ||
          phase === "completed"
        ? 100
        : 0;

  if (phase === "completed") {
    return createTokenBlueprintCreateProgress(
      createCompletedTokenBlueprintProgress({
        completedUploadCount,
        expectedUploadCount,
        title: "保存が完了しました",
        message:
          "トークン設計の保存が完了しました。",
      }),
      "completed",
      {
        operationId: operation.id,
        tokenBlueprintId:
          operation.tokenBlueprintId,
        percentage: 100,
        retryCount: operation.retryCount,
        maxRetries: operation.maxRetries,
      },
    );
  }

  if (phase === "failed_retryable") {
    return createTokenBlueprintCreateProgress(
      createFailedTokenBlueprintProgress(
        operation.lastError || "",
        {
          title: "保存処理に失敗しました",
          message:
            "一時的なエラーが発生しました。再試行できます。",
        },
      ),
      "failed_retryable",
      {
        operationId: operation.id,
        tokenBlueprintId:
          operation.tokenBlueprintId,
        percentage: uploadPercentage,
        completedUploadCount,
        expectedUploadCount,
        retryCount: operation.retryCount,
        maxRetries: operation.maxRetries,
        canRetry: true,
        isTerminal: false,
      },
    );
  }

  if (phase === "failed_fatal") {
    return createTokenBlueprintCreateProgress(
      createFailedTokenBlueprintProgress(
        operation.lastError || "",
        {
          title:
            "トークン設計を保存できませんでした",
          message:
            "保存処理を継続できないエラーが発生しました。",
        },
      ),
      "failed_fatal",
      {
        operationId: operation.id,
        tokenBlueprintId:
          operation.tokenBlueprintId,
        percentage: uploadPercentage,
        completedUploadCount,
        expectedUploadCount,
        retryCount: operation.retryCount,
        maxRetries: operation.maxRetries,
      },
    );
  }

  if (phase === "queued") {
    return createTokenBlueprintCreateProgress(
      createInitialTokenBlueprintProgress(),
      "queued",
      {
        operationId: operation.id,
        tokenBlueprintId:
          operation.tokenBlueprintId,
        title:
          "ファイル転送完了・保存準備中",
        message:
          "ファイル転送は完了しました。保存処理を開始します。",
        percentage: 100,
        completedUploadCount,
        expectedUploadCount,
        retryCount: operation.retryCount,
        maxRetries: operation.maxRetries,
        isBrowserDependent: false,
        isBlockingNavigation: false,
        isTerminal: false,
      },
    );
  }

  if (phase === "processing") {
    return createTokenBlueprintCreateProgress(
      createInitialTokenBlueprintProgress(),
      "processing",
      {
        operationId: operation.id,
        tokenBlueprintId:
          operation.tokenBlueprintId,
        title: "トークン設計を保存中",
        message:
          "サーバーでトークン設計を保存しています。この画面を離れても処理は継続します。",
        percentage: 100,
        completedUploadCount,
        expectedUploadCount,
        retryCount: operation.retryCount,
        maxRetries: operation.maxRetries,
        isBrowserDependent: false,
        isBlockingNavigation: false,
        isTerminal: false,
      },
    );
  }

  return createTokenBlueprintCreateProgress(
    createInitialTokenBlueprintProgress(),
    "uploading",
    {
      operationId: operation.id,
      tokenBlueprintId:
        operation.tokenBlueprintId,
      title: "ファイルを転送中",
      message:
        "ファイル転送が完了するまで、この画面を閉じたり移動したりしないでください。",
      percentage: uploadPercentage,
      completedUploadCount,
      expectedUploadCount,
      retryCount: operation.retryCount,
      maxRetries: operation.maxRetries,
      isBrowserDependent: true,
      isBlockingNavigation: true,
      isTerminal: false,
    },
  );
}

export function isTokenBlueprintCreateBrowserUploadPhase(
  progress: TokenBlueprintCreateProgress,
): boolean {
  return progress.phase === "uploading";
}

export function shouldBlockTokenBlueprintCreateNavigation(
  progress: TokenBlueprintCreateProgress,
): boolean {
  return progress.isBlockingNavigation;
}

export function isTokenBlueprintCreateOperationPollingRequired(
  progress: TokenBlueprintCreateProgress,
): boolean {
  return (
    progress.phase === "queued" ||
    progress.phase === "processing"
  );
}

export function isTokenBlueprintCreateOperationFinished(
  progress: TokenBlueprintCreateProgress,
): boolean {
  return (
    progress.phase === "completed" ||
    progress.phase === "failed_fatal"
  );
}