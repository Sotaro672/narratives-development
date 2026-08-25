// frontend/console/shell/src/features/tokenBlueprint/presentation/model/tokenBlueprintCreateProgress.ts

import type {
  TokenBlueprintCreateOperation,
  TokenBlueprintCreateOperationStatus,
} from "../../infrastructure/repository/tokenBlueprintCreateOperationRepositoryHTTP";
import {
  createCompletedTokenBlueprintProgress,
  createFailedTokenBlueprintProgress,
  createInitialTokenBlueprintProgress,
  createUploadingTokenBlueprintProgress as createUploadingBaseProgress,
  type TokenBlueprintProgress,
  type TokenBlueprintProgressTarget,
} from "./tokenBlueprintProgress";

export type TokenBlueprintCreateProgressPhase =
  | "idle"
  | "starting"
  | "uploading"
  | "queued"
  | "processing"
  | "completed"
  | "failed_retryable"
  | "failed_fatal";

export type TokenBlueprintCreateUploadTarget = TokenBlueprintProgressTarget;

export type TokenBlueprintCreateProgress = Omit<TokenBlueprintProgress, "phase"> & {
  phase: TokenBlueprintCreateProgressPhase;
  operationId?: string;
  tokenBlueprintId?: string;
  retryCount: number;
  maxRetries: number;
  canRetry: boolean;
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

function clampPercentage(value: number): number {
  if (!Number.isFinite(value)) {
    return 0;
  }

  return Math.min(100, Math.max(0, Math.round(value)));
}

function normalizeCount(value: number): number {
  if (!Number.isFinite(value)) {
    return 0;
  }

  return Math.max(0, Math.floor(value));
}

function operationStatusToPhase(
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

function createCreateProgress(
  base: TokenBlueprintProgress,
  phase: TokenBlueprintCreateProgressPhase,
  overrides?: Partial<Omit<TokenBlueprintCreateProgress, "phase">>,
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
  return createCreateProgress(
    createInitialTokenBlueprintProgress(),
    "idle",
  );
}

export function createStartingTokenBlueprintCreateProgress(): TokenBlueprintCreateProgress {
  return createCreateProgress(
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
  const base = createUploadingBaseProgress({
    target: input.target,
    fileName: input.fileName,
    transferredBytes: input.transferredBytes,
    totalBytes: input.totalBytes,
    completedUploadCount: input.completedUploadCount,
    expectedUploadCount: input.expectedUploadCount,
  });

  return createCreateProgress(base, "uploading", {
    operationId: input.operation?.id,
    tokenBlueprintId: input.operation?.tokenBlueprintId,
    retryCount: input.operation?.retryCount ?? 0,
    maxRetries: input.operation?.maxRetries ?? 0,
  });
}

export function createTokenBlueprintCreateProgressFromOperation(
  operation: TokenBlueprintCreateOperation,
): TokenBlueprintCreateProgress {
  const phase = operationStatusToPhase(operation.status);
  const completedUploadCount = normalizeCount(operation.completedUploadCount);
  const expectedUploadCount = normalizeCount(operation.expectedUploadCount);

  const uploadPercentage =
    expectedUploadCount > 0
      ? clampPercentage((completedUploadCount / expectedUploadCount) * 100)
      : phase === "queued" || phase === "processing" || phase === "completed"
        ? 100
        : 0;

  if (phase === "completed") {
    return createCreateProgress(
      createCompletedTokenBlueprintProgress({
        completedUploadCount,
        expectedUploadCount,
        title: "保存が完了しました",
        message: "トークン設計の保存が完了しました。",
      }),
      "completed",
      {
        operationId: operation.id,
        tokenBlueprintId: operation.tokenBlueprintId,
        percentage: 100,
        retryCount: operation.retryCount,
        maxRetries: operation.maxRetries,
      },
    );
  }

  if (phase === "failed_retryable") {
    return createCreateProgress(
      createFailedTokenBlueprintProgress(
        operation.lastError || "",
        {
          title: "保存処理に失敗しました",
          message: "一時的なエラーが発生しました。再試行できます。",
        },
      ),
      "failed_retryable",
      {
        operationId: operation.id,
        tokenBlueprintId: operation.tokenBlueprintId,
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
    return createCreateProgress(
      createFailedTokenBlueprintProgress(
        operation.lastError || "",
        {
          title: "トークン設計を保存できませんでした",
          message: "保存処理を継続できないエラーが発生しました。",
        },
      ),
      "failed_fatal",
      {
        operationId: operation.id,
        tokenBlueprintId: operation.tokenBlueprintId,
        percentage: uploadPercentage,
        completedUploadCount,
        expectedUploadCount,
        retryCount: operation.retryCount,
        maxRetries: operation.maxRetries,
      },
    );
  }

  if (phase === "queued") {
    return createCreateProgress(
      createInitialTokenBlueprintProgress(),
      "queued",
      {
        operationId: operation.id,
        tokenBlueprintId: operation.tokenBlueprintId,
        title: "ファイル転送完了・保存準備中",
        message: "ファイル転送は完了しました。保存処理を開始します。",
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
    return createCreateProgress(
      createInitialTokenBlueprintProgress(),
      "processing",
      {
        operationId: operation.id,
        tokenBlueprintId: operation.tokenBlueprintId,
        title: "トークン設計を保存中",
        message: "サーバーでトークン設計を保存しています。この画面を離れても処理は継続します。",
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

  return createCreateProgress(
    createInitialTokenBlueprintProgress(),
    "uploading",
    {
      operationId: operation.id,
      tokenBlueprintId: operation.tokenBlueprintId,
      title: "ファイルを転送中",
      message: "ファイル転送が完了するまで、この画面を閉じたり移動したりしないでください。",
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
  return progress.phase === "queued" || progress.phase === "processing";
}

export function isTokenBlueprintCreateOperationFinished(
  progress: TokenBlueprintCreateProgress,
): boolean {
  return progress.phase === "completed" || progress.phase === "failed_fatal";
}