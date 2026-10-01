// frontend/mall/src/features/shared/types/chatComposer.ts

export type ChatComposerConfig = {
  content: string;
  placeholder?: string;
  files?: File[];
  error?: string | null;
  submitting?: boolean;
  canSubmit: boolean;
  disabled?: boolean;
  submitLabel?: string;
  submittingLabel?: string;
  maxLength?: number | null;
  maxFiles?: number;
  accept?: string;
  onContentChange: (value: string) => void;
  onFilesAdd?: (files: File[]) => void;
  onRemoveFile?: (index: number) => void;
  onSubmit: () => void | Promise<void>;
};