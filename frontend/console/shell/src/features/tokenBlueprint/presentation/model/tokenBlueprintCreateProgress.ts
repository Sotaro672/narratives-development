// frontend/console/shell/src/features/tokenBlueprint/presentation/model/tokenBlueprintCreateProgress.ts

import type {
  TokenBlueprintCreateOperation,
  TokenBlueprintCreateOperationStatus,
} from "../../infrastructure/repository/tokenBlueprintCreateOperationRepositoryHTTP";

export type TokenBlueprintCreateProgressPhase =
  | "idle"
  | "starting"
  | "uploading"
  | "queued"
  | "processing"
  | "completed"
  | "failed_retryable"
  | "failed_fatal";

export type TokenBlueprintCreateUploadTarget =
  | "icon"
  | "content";

export type TokenBlueprintCreateProgress = {
  phase: TokenBlueprintCreateProgressPhase;

  operationId?: string;

  tokenBlueprintId?: string;

  title: string;

  message: string;

  percentage: number;

  transferredBytes: number;

  totalBytes: number;

  currentFileName?: string;

  currentUploadTarget?: TokenBlueprintCreateUploadTarget;

  completedUploadCount: number;

  expectedUploadCount: number;

  retryCount: number;

  maxRetries: number;

  errorMessage?: string;

  isBrowserDependent: boolean;

  isBlockingNavigation: boolean;

  canRetry: boolean;

  isTerminal: boolean;
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

function clampPercentage(
  value: number,
): number {
  if (!Number.isFinite(value)) {
    return 0;
  }

  return Math.min(
    100,
    Math.max(
      0,
      Math.round(value),
    ),
  );
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

function normalizeByteCount(
  value: number,
): number {
  if (!Number.isFinite(value)) {
    return 0;
  }

  return Math.max(
    0,
    value,
  );
}

function normalizeCount(
  value: number,
): number {
  if (!Number.isFinite(value)) {
    return 0;
  }

  return Math.max(
    0,
    Math.floor(value),
  );
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

function phaseTitle(
  phase: TokenBlueprintCreateProgressPhase,
): string {
  switch (phase) {
    case "idle":
      return "";

    case "starting":
      return "作成準備中";

    case "uploading":
      return "ファイルを転送中";

    case "queued":
      return "ファイル転送完了・保存準備中";

    case "processing":
      return "トークン設計を保存中";

    case "completed":
      return "保存が完了しました";

    case "failed_retryable":
      return "保存処理に失敗しました";

    case "failed_fatal":
      return "トークン設計を保存できませんでした";
  }
}

function phaseMessage(
  phase: TokenBlueprintCreateProgressPhase,
): string {
  switch (phase) {
    case "idle":
      return "";

    case "starting":
      return "トークン設計の作成準備をしています。";

    case "uploading":
      return "ファイル転送が完了するまで、この画面を閉じたり移動したりしないでください。";

    case "queued":
      return "ファイル転送は完了しました。保存処理を開始します。";

    case "processing":
      return "サーバーでトークン設計を保存しています。この画面を離れても処理は継続します。";

    case "completed":
      return "トークン設計の保存が完了しました。";

    case "failed_retryable":
      return "一時的なエラーが発生しました。再試行できます。";

    case "failed_fatal":
      return "保存処理を継続できないエラーが発生しました。";
  }
}

function buildBaseProgress(
  phase: TokenBlueprintCreateProgressPhase,
): TokenBlueprintCreateProgress {
  return {
    phase,

    title: phaseTitle(phase),

    message: phaseMessage(phase),

    percentage: 0,

    transferredBytes: 0,

    totalBytes: 0,

    completedUploadCount: 0,

    expectedUploadCount: 0,

    retryCount: 0,

    maxRetries: 0,

    isBrowserDependent:
      phase === "uploading",

    isBlockingNavigation:
      phase === "uploading",

    canRetry:
      phase === "failed_retryable",

    isTerminal:
      phase === "completed" ||
      phase === "failed_fatal",
  };
}

export function createInitialTokenBlueprintCreateProgress(): TokenBlueprintCreateProgress {
  return buildBaseProgress(
    "idle",
  );
}

export function createStartingTokenBlueprintCreateProgress(): TokenBlueprintCreateProgress {
  return buildBaseProgress(
    "starting",
  );
}

export function createUploadingTokenBlueprintCreateProgress(
  input: TokenBlueprintCreateUploadProgressInput,
): TokenBlueprintCreateProgress {
  const transferredBytes =
    normalizeByteCount(
      input.transferredBytes,
    );

  const totalBytes =
    normalizeByteCount(
      input.totalBytes,
    );

  const completedUploadCount =
    normalizeCount(
      input.completedUploadCount,
    );

  const expectedUploadCount =
    normalizeCount(
      input.expectedUploadCount,
    );

  return {
    ...buildBaseProgress(
      "uploading",
    ),

    operationId:
      input.operation?.id,

    tokenBlueprintId:
      input.operation?.tokenBlueprintId,

    percentage:
      calculatePercentage(
        transferredBytes,
        totalBytes,
      ),

    transferredBytes,

    totalBytes,

    currentFileName:
      input.fileName,

    currentUploadTarget:
      input.target,

    completedUploadCount,

    expectedUploadCount,

    retryCount:
      input.operation?.retryCount ?? 0,

    maxRetries:
      input.operation?.maxRetries ?? 0,
  };
}

export function createTokenBlueprintCreateProgressFromOperation(
  operation: TokenBlueprintCreateOperation,
): TokenBlueprintCreateProgress {
  const phase =
    operationStatusToPhase(
      operation.status,
    );

  const completedUploadCount =
    normalizeCount(
      operation.completedUploadCount,
    );

  const expectedUploadCount =
    normalizeCount(
      operation.expectedUploadCount,
    );

  const uploadPercentage =
    expectedUploadCount > 0
      ? clampPercentage(
          (
            completedUploadCount /
            expectedUploadCount
          ) * 100,
        )
      : phase === "queued" ||
          phase === "processing" ||
          phase === "completed"
        ? 100
        : 0;

  return {
    ...buildBaseProgress(
      phase,
    ),

    operationId:
      operation.id,

    tokenBlueprintId:
      operation.tokenBlueprintId,

    percentage:
      phase === "queued" ||
      phase === "processing" ||
      phase === "completed"
        ? 100
        : uploadPercentage,

    completedUploadCount,

    expectedUploadCount,

    retryCount:
      operation.retryCount,

    maxRetries:
      operation.maxRetries,

    errorMessage:
      operation.lastError || undefined,
  };
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