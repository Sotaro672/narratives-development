// frontend/console/shell/src/features/list/presentation/modal/listProgress.ts 
 
export type ListProgressPhase = 
  | "idle" 
  | "preparing" 
  | "uploading" 
  | "saving" 
  | "completed" 
  | "failed"; 
 
export type ListProgress = { 
  phase: ListProgressPhase; 
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
 
export type ListUploadProgressInput = { 
  fileName: string; 
  transferredBytes: number; 
  totalBytes: number; 
  completedUploadCount: number; 
  expectedUploadCount: number; 
  title?: string; 
  message?: string; 
}; 
 
export type ListSavingProgressInput = { 
  completedUploadCount?: number; 
  expectedUploadCount?: number; 
  transferredBytes?: number; 
  totalBytes?: number; 
  title?: string; 
  message?: string; 
}; 
 
export type ListCompletedProgressInput = { 
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
  phase: ListProgressPhase, 
): string { 
  switch (phase) { 
    case "idle": 
      return ""; 
 
    case "preparing": 
      return "保存準備中"; 
 
    case "uploading": 
      return "画像を転送中"; 
 
    case "saving": 
      return "出品を保存中"; 
 
    case "completed": 
      return "保存が完了しました"; 
 
    case "failed": 
      return "保存に失敗しました"; 
  } 
} 
 
function defaultMessage( 
  phase: ListProgressPhase, 
): string { 
  switch (phase) { 
    case "idle": 
      return ""; 
 
    case "preparing": 
      return "出品情報の保存準備をしています。"; 
 
    case "uploading": 
      return "画像転送が完了するまで、この画面を閉じたり移動したりしないでください。"; 
 
    case "saving": 
      return "画像転送が完了しました。出品情報を保存しています。"; 
 
    case "completed": 
      return "出品情報の保存が完了しました。"; 
 
    case "failed": 
      return "出品情報の保存中にエラーが発生しました。"; 
  } 
} 
 
function buildBaseProgress( 
  phase: ListProgressPhase, 
): ListProgress { 
  const browserDependent = 
    phase === "uploading"; 
 
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
 
export function createInitialListProgress(): ListProgress { 
  return buildBaseProgress("idle"); 
} 
 
export function createPreparingListProgress( 
  input?: { 
    title?: string; 
    message?: string; 
  }, 
): ListProgress { 
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
 
export function createUploadingListProgress( 
  input: ListUploadProgressInput, 
): ListProgress { 
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
    percentage: 
      calculatePercentage( 
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
 
export function createSavingListProgress( 
  input?: ListSavingProgressInput, 
): ListProgress { 
  const transferredBytes = 
    normalizeByteCount(input?.transferredBytes ?? 0); 
 
  const totalBytes = 
    normalizeByteCount(input?.totalBytes ?? 0); 
 
  const completedUploadCount = 
    normalizeCount(input?.completedUploadCount ?? 0); 
 
  const expectedUploadCount = 
    normalizeCount(input?.expectedUploadCount ?? 0); 
 
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
 
export function createCompletedListProgress( 
  input?: ListCompletedProgressInput, 
): ListProgress { 
  const transferredBytes = 
    normalizeByteCount(input?.transferredBytes ?? 0); 
 
  const totalBytes = 
    normalizeByteCount(input?.totalBytes ?? 0); 
 
  const completedUploadCount = 
    normalizeCount(input?.completedUploadCount ?? 0); 
 
  const expectedUploadCount = 
    normalizeCount(input?.expectedUploadCount ?? 0); 
 
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
 
export function createFailedListProgress( 
  errorMessage: string, 
  input?: { 
    title?: string; 
    message?: string; 
  }, 
): ListProgress { 
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
 
export function shouldBlockListNavigation( 
  progress: ListProgress, 
): boolean { 
  return progress.phase === "uploading"; 
} 
 
export function isListProgressVisible( 
  progress: ListProgress, 
): boolean { 
  return progress.phase !== "idle"; 
} 
 
export function isListProgressFinished( 
  progress: ListProgress, 
): boolean { 
  return progress.isTerminal; 
}