// frontend/console/shell/src/features/tokenBlueprint/presentation/hook/useTokenBlueprintCreate.tsx

import * as React from "react";
import { useNavigate } from "react-router-dom";

import { useAssigneeSelection } from "../../../admin/presentation/hook/useAssigneeSelection";
import {
  createTokenBlueprintCreateOperationIdempotencyKey,
  createTokenBlueprintWithOptionalIcon,
  type CreateTokenBlueprintInput,
} from "../../application/tokenBlueprintCreateService";
import {
  fetchTokenBlueprintCreateOperation,
  type TokenBlueprintCreateOperation,
} from "../../infrastructure/repository/tokenBlueprintCreateOperationRepositoryHTTP";
import {
  createInitialTokenBlueprintCreateProgress,
  createStartingTokenBlueprintCreateProgress,
  createTokenBlueprintCreateProgressFromOperation,
  createUploadingTokenBlueprintCreateProgress,
  type TokenBlueprintCreateProgress,
} from "../model/tokenBlueprintCreateProgress";

const CREATE_OPERATION_ID_KEY = "tokenBlueprint.create.operationId";
const CREATE_IDEMPOTENCY_KEY = "tokenBlueprint.create.idempotencyKey";
const CREATE_OPERATION_POLL_INTERVAL_MS = 1500;

export type SaveTokenBlueprintInput = Omit<
  CreateTokenBlueprintInput,
  | "idempotencyKey"
  | "operationId"
  | "onOperationChange"
  | "onIconProgress"
  | "onContentProgress"
>;

function readSessionValue(key: string): string {
  try {
    return sessionStorage.getItem(key) ?? "";
  } catch {
    return "";
  }
}

function writeSessionValue(key: string, value: string): void {
  try {
    sessionStorage.setItem(key, value);
  } catch {
    // sessionStorage unavailable.
  }
}

function removeSessionValue(key: string): void {
  try {
    sessionStorage.removeItem(key);
  } catch {
    // sessionStorage unavailable.
  }
}

function clearCreateOperationSession(): void {
  removeSessionValue(CREATE_OPERATION_ID_KEY);
  removeSessionValue(CREATE_IDEMPOTENCY_KEY);
}

function calculateTotalUploadBytes(input: SaveTokenBlueprintInput): number {
  const iconSize = input.iconFile?.size ?? 0;
  const contentSize = (input.contents ?? []).reduce(
    (total, content) => total + content.file.size,
    0,
  );
  return iconSize + contentSize;
}

function calculateUploadedBytes(
  operation: TokenBlueprintCreateOperation,
  input: SaveTokenBlueprintInput,
): number {
  let uploadedBytes = 0;

  if (operation.icon?.uploaded && input.iconFile) {
    uploadedBytes += input.iconFile.size;
  }

  const contentById = new Map(
    (input.contents ?? []).map((content) => [content.id, content]),
  );

  for (const content of operation.contents) {
    if (!content.uploaded) continue;

    const localContent = contentById.get(content.id);
    if (localContent) uploadedBytes += localContent.file.size;
  }

  return uploadedBytes;
}

function shouldPollOperation(status: TokenBlueprintCreateOperation["status"]): boolean {
  return (
    status === "queued" ||
    status === "processing" ||
    status === "failed_retryable"
  );
}

/**
 * TokenBlueprint作成ページ用ロジック。
 *
 * Create Operationを正として、
 * browser依存uploadとCloud Tasks処理を分離する。
 *
 * waiting_upload:
 * - local Fileが必要
 * - upload実行中のみ画面離脱を禁止
 *
 * queued / processing:
 * - browserから独立済み
 * - Operationをpolling
 *
 * completed:
 * - sessionStorageを削除
 * - TokenBlueprint詳細へ遷移
 */
export function useTokenBlueprintCreate() {
  const navigate = useNavigate();

  const {
    assigneeId,
    assigneeName,
    assigneeCandidates,
    loadingMembers,
    handleSelectAssignee,
  } = useAssigneeSelection({ defaultToCurrentMember: true });

  const [operation, setOperation] = React.useState<TokenBlueprintCreateOperation | null>(null);
  const operationRef = React.useRef<TokenBlueprintCreateOperation | null>(null);

  const [progress, setProgress] = React.useState<TokenBlueprintCreateProgress>(
    createInitialTokenBlueprintCreateProgress,
  );
  const [saving, setSaving] = React.useState(false);
  const [createError, setCreateError] = React.useState<string | null>(null);

  const isUploading = progress.phase === "uploading" && saving;
  const progressOpen = progress.phase !== "idle";

  const updateOperation = React.useCallback((next: TokenBlueprintCreateOperation) => {
    operationRef.current = next;
    setOperation(next);
    writeSessionValue(CREATE_OPERATION_ID_KEY, next.id);

    if (next.status !== "waiting_upload") {
      setProgress(createTokenBlueprintCreateProgressFromOperation(next));
      return;
    }

    if (next.expectedUploadCount > 0) {
      setProgress((current) =>
        current.phase === "starting"
          ? createTokenBlueprintCreateProgressFromOperation(next)
          : current,
      );
    }
  }, []);

  React.useEffect(() => {
    if (!isUploading) return;

    const handleBeforeUnload = (event: BeforeUnloadEvent) => {
      event.preventDefault();
      event.returnValue = "";
    };

    window.addEventListener("beforeunload", handleBeforeUnload);
    return () => window.removeEventListener("beforeunload", handleBeforeUnload);
  }, [isUploading]);

  React.useEffect(() => {
    const operationId = readSessionValue(CREATE_OPERATION_ID_KEY);
    if (!operationId) return;

    let cancelled = false;

    void (async () => {
      try {
        const restored = await fetchTokenBlueprintCreateOperation(operationId);
        if (cancelled) return;

        operationRef.current = restored;
        setOperation(restored);

        if (restored.status === "completed") {
          clearCreateOperationSession();
          setProgress(createTokenBlueprintCreateProgressFromOperation(restored));
          navigate(`/tokenBlueprint/${encodeURIComponent(restored.tokenBlueprintId)}`, {
            replace: true,
          });
          return;
        }

        if (restored.status === "waiting_upload") {
          // Reload後はlocal Fileを復元できないため、
          // 実際にuploadを再開するまでは画面離脱を禁止しない。
          setProgress(createInitialTokenBlueprintCreateProgress());
          return;
        }

        setProgress(createTokenBlueprintCreateProgressFromOperation(restored));
      } catch {
        // 一時的な取得失敗ではsessionを削除しない。
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [navigate]);

  React.useEffect(() => {
    if (!operation || !shouldPollOperation(operation.status)) return;

    let cancelled = false;

    const poll = async () => {
      try {
        const next = await fetchTokenBlueprintCreateOperation(operation.id);
        if (cancelled) return;

        operationRef.current = next;
        setOperation(next);
        setProgress(createTokenBlueprintCreateProgressFromOperation(next));

        if (next.status === "completed") {
          clearCreateOperationSession();
          navigate(`/tokenBlueprint/${encodeURIComponent(next.tokenBlueprintId)}`, {
            replace: true,
          });
        }
      } catch {
        // polling中の一時的な通信失敗では状態を変更しない。
      }
    };

    const timer = window.setInterval(
      () => void poll(),
      CREATE_OPERATION_POLL_INTERVAL_MS,
    );

    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, [operation, navigate]);

  const onBack = React.useCallback(() => {
    if (isUploading) return;
    navigate("/tokenBlueprint", { replace: true });
  }, [isUploading, navigate]);

  const onCloseProgress = React.useCallback(() => {
    if (progress.isBlockingNavigation) return;

    if (
      progress.phase === "failed_retryable" ||
      progress.phase === "failed_fatal"
    ) {
      setProgress(createInitialTokenBlueprintCreateProgress());
    }
  }, [progress]);

  const onSave = React.useCallback(
    async (input: SaveTokenBlueprintInput): Promise<TokenBlueprintCreateOperation> => {
      if (!input.assigneeId) {
        throw new Error("assigneeId is required");
      }

      if (saving) {
        throw new Error("token blueprint create operation is already running");
      }

      setSaving(true);
      setCreateError(null);
      setProgress(createStartingTokenBlueprintCreateProgress());

      let idempotencyKey = readSessionValue(CREATE_IDEMPOTENCY_KEY);
      if (!idempotencyKey) {
        idempotencyKey = createTokenBlueprintCreateOperationIdempotencyKey();
        writeSessionValue(CREATE_IDEMPOTENCY_KEY, idempotencyKey);
      }

      const savedOperationId = readSessionValue(CREATE_OPERATION_ID_KEY);
      const totalUploadBytes = calculateTotalUploadBytes(input);

      try {
        const result = await createTokenBlueprintWithOptionalIcon({
          ...input,
          idempotencyKey,
          operationId: savedOperationId || undefined,

          onOperationChange: (next) => {
            updateOperation(next);
          },

          onIconProgress: (uploadProgress) => {
            const currentOperation = operationRef.current;
            if (!currentOperation) return;

            const alreadyUploadedBytes = calculateUploadedBytes(
              currentOperation,
              input,
            );

            setProgress(
              createUploadingTokenBlueprintCreateProgress({
                operation: currentOperation,
                target: "icon",
                fileName: input.iconFile?.name ?? "",
                transferredBytes:
                  alreadyUploadedBytes + uploadProgress.transferredBytes,
                totalBytes: totalUploadBytes,
                completedUploadCount: currentOperation.completedUploadCount,
                expectedUploadCount: currentOperation.expectedUploadCount,
              }),
            );
          },

          onContentProgress: (uploadProgress) => {
            const currentOperation = operationRef.current;
            if (!currentOperation) return;

            const alreadyUploadedBytes = Math.max(
              0,
              totalUploadBytes - uploadProgress.totalBytes,
            );

            const alreadyCompletedCount = Math.max(
              0,
              currentOperation.expectedUploadCount - uploadProgress.totalCount,
            );

            setProgress(
              createUploadingTokenBlueprintCreateProgress({
                operation: currentOperation,
                target: "content",
                fileName: uploadProgress.fileName,
                transferredBytes:
                  alreadyUploadedBytes + uploadProgress.transferredBytes,
                totalBytes: totalUploadBytes,
                completedUploadCount:
                  alreadyCompletedCount + uploadProgress.completedCount,
                expectedUploadCount: currentOperation.expectedUploadCount,
              }),
            );
          },
        });

        operationRef.current = result;
        setOperation(result);
        setProgress(createTokenBlueprintCreateProgressFromOperation(result));

        if (result.status === "completed") {
          clearCreateOperationSession();
          navigate(`/tokenBlueprint/${encodeURIComponent(result.tokenBlueprintId)}`, {
            replace: true,
          });
        }

        return result;
      } catch (error) {
        const message =
          error instanceof Error
            ? error.message
            : "トークン設計の作成に失敗しました。";

        setCreateError(message);

        // Storage upload失敗時にuploadingのまま残すと
        // navigation guardを解除できなくなるためidleへ戻す。
        setProgress(createInitialTokenBlueprintCreateProgress());

        throw error;
      } finally {
        setSaving(false);
      }
    },
    [navigate, saving, updateOperation],
  );

  const initialTokenBlueprint = React.useMemo(
    () => ({
      id: "",
      name: "",
      symbol: "",
      brandId: "",
      brandName: "",
      description: "",
      assigneeId,
      minted: false,
    }),
    [assigneeId],
  );

  const onEditAssignee = React.useCallback(() => {}, []);
  const onClickAssignee = React.useCallback(() => {}, []);

  return {
    initialTokenBlueprint,

    assigneeId,
    assigneeName,
    assigneeCandidates,
    loadingMembers,
    onSelectAssignee: handleSelectAssignee,
    onEditAssignee,
    onClickAssignee,

    operation,
    progress,
    progressOpen,
    saving,
    isUploading,
    createError,
    onCloseProgress,

    initialEditMode: true,

    onBack,
    onSave,
  };
}