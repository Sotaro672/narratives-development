// frontend/console/shell/src/features/announcement/presentation/components/inputCard.tsx

import { useEffect, useMemo, useRef, useState } from "react";
import type * as React from "react";

import type {
  AnnouncementInputAttachment,
  AnnouncementInputPayload,
} from "../../application/announcement_input";

import { Button } from "../../../../shared/ui/button";
import DeleteButton from "../../../../shared/ui/delete";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";

export type InputCardMode = "view" | "edit";

type Props = {
  title?: string;
  mode?: InputCardMode;
  initialTitle?: string;
  initialText?: string;
  initialAttachments?: AnnouncementInputAttachment[];
  saving?: boolean;
  sending?: boolean;
  onChange?: (payload: AnnouncementInputPayload) => void;
};

type PreviewImage = {
  key: string;
  url: string;
  name: string;
  revokeOnCleanup: boolean;
};

const EMPTY_INITIAL_ATTACHMENTS: AnnouncementInputAttachment[] = [];

function fileKey(file: File, index: number): string {
  return `${file.name}-${file.size}-${file.lastModified}-${index}`;
}

function getFileIdentity(file: File): string {
  return `${file.name}-${file.size}-${file.lastModified}`;
}

function ImageIcon() {
  return (
    <svg width="28" height="28" viewBox="0 0 24 24" fill="none">
      <path
        d="M21 19V5a2 2 0 0 0-2-2H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2Z"
        stroke="currentColor"
        strokeWidth="1.6"
      />
      <path
        d="M8.5 10.5a1.5 1.5 0 1 0 0-3 1.5 1.5 0 0 0 0 3Z"
        stroke="currentColor"
        strokeWidth="1.6"
      />
      <path
        d="M21 16l-5.5-5.5a2 2 0 0 0-2.8 0L5 18"
        stroke="currentColor"
        strokeWidth="1.6"
      />
    </svg>
  );
}

function PlusIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none">
      <path
        d="M12 5v14M5 12h14"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
      />
    </svg>
  );
}

function formatViewText(value: string): string {
  return value.trim() || "-";
}

export default function InputCard({
  title = "入力",
  mode = "edit",
  initialTitle = "",
  initialText = "",
  initialAttachments = EMPTY_INITIAL_ATTACHMENTS,
  saving = false,
  sending = false,
  onChange,
}: Props) {
  const [inputTitle, setInputTitle] = useState(initialTitle);
  const [text, setText] = useState(initialText);
  const [attachments, setAttachments] =
    useState<AnnouncementInputAttachment[]>(initialAttachments);
  const [mainImageIndex, setMainImageIndex] = useState(0);

  const imageInputRef = useRef<HTMLInputElement | null>(null);

  const isEditMode = mode === "edit";
  const isViewMode = mode === "view";
  const isBusy = saving || sending;
  const isDisabled = isBusy || isViewMode;

  useEffect(() => {
    setInputTitle(initialTitle);
  }, [initialTitle]);

  useEffect(() => {
    setText(initialText);
  }, [initialText]);

  useEffect(() => {
    setAttachments(initialAttachments);
    setMainImageIndex(0);
  }, [initialAttachments]);

  useEffect(() => {
    onChange?.({
      title: inputTitle,
      text,
      attachments,
    });
  }, [inputTitle, text, attachments, onChange]);

  const previewImages = useMemo<PreviewImage[]>(() => {
    return attachments.map((attachment, index) => {
      if (attachment.type === "new") {
        return {
          key: fileKey(attachment.file, index),
          url: URL.createObjectURL(attachment.file),
          name: attachment.file.name,
          revokeOnCleanup: true,
        };
      }

      return {
        key: attachment.id,
        url: attachment.fileUrl,
        name: attachment.fileName,
        revokeOnCleanup: false,
      };
    });
  }, [attachments]);

  useEffect(() => {
    return () => {
      previewImages.forEach((item) => {
        if (item.revokeOnCleanup) {
          URL.revokeObjectURL(item.url);
        }
      });
    };
  }, [previewImages]);

  useEffect(() => {
    if (attachments.length === 0) {
      if (mainImageIndex !== 0) {
        setMainImageIndex(0);
      }
      return;
    }

    if (mainImageIndex > attachments.length - 1) {
      setMainImageIndex(attachments.length - 1);
    }
  }, [attachments, mainImageIndex]);

  const hasImages = previewImages.length > 0;
  const mainImage = previewImages[mainImageIndex] ?? null;
  const thumbIndices = previewImages
    .map((_, index) => index)
    .filter((index) => index !== mainImageIndex);

  const openPicker = () => {
    if (!isEditMode || isBusy) return;
    imageInputRef.current?.click();
  };

  const addImages = (nextFiles: File[]) => {
    if (!isEditMode || nextFiles.length === 0) return;

    setAttachments((previousAttachments) => {
      const existingFileIdentities = previousAttachments
        .filter((attachment) => attachment.type === "new")
        .map((attachment) => getFileIdentity(attachment.file));

      const seen = new Set(existingFileIdentities);
      const merged = [...previousAttachments];
      let firstAddedIndex = -1;

      for (const file of nextFiles) {
        if (!file.type.startsWith("image/")) continue;

        const identity = getFileIdentity(file);
        if (seen.has(identity)) continue;

        seen.add(identity);

        if (firstAddedIndex === -1) {
          firstAddedIndex = merged.length;
        }

        merged.push({
          type: "new",
          file,
        });
      }

      if (firstAddedIndex !== -1) {
        setMainImageIndex(firstAddedIndex);
      }

      return merged;
    });
  };

  const handleSelectImages = (
    event: React.ChangeEvent<HTMLInputElement>,
  ) => {
    const nextFiles = Array.from(event.target.files ?? []);
    addImages(nextFiles);
    event.target.value = "";
  };

  const handleDropImages = (
    event: React.DragEvent<HTMLDivElement>,
  ) => {
    event.preventDefault();
    event.stopPropagation();

    if (!isEditMode || isBusy) return;

    const nextFiles = Array.from(event.dataTransfer.files ?? []);
    addImages(nextFiles);
  };

  const handleDragOverImages = (
    event: React.DragEvent<HTMLDivElement>,
  ) => {
    event.preventDefault();
    event.stopPropagation();
  };

  const handleRemoveImageAt = (targetIndex: number) => {
    if (!isEditMode || isBusy) return;

    setAttachments((previousAttachments) =>
      previousAttachments.filter((_, index) => index !== targetIndex),
    );

    setMainImageIndex((previousIndex) => {
      if (targetIndex < previousIndex) return previousIndex - 1;
      if (targetIndex === previousIndex) return 0;
      return previousIndex;
    });
  };

  const handleClearImages = () => {
    if (!isEditMode || isBusy) return;

    setAttachments([]);
    setMainImageIndex(0);
  };

  const handleSelectMainImage = (index: number) => {
    if (!isEditMode) return;
    setMainImageIndex(index);
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
      </CardHeader>

      <CardContent>
        <div className="space-y-4">
          <div className="space-y-2">
            <div className="flex items-center justify-between gap-3">
              <label className="text-sm font-medium text-slate-700">
                画像アップロード
              </label>

              {isEditMode && hasImages && (
                <Button
                  type="button"
                  variant="ghost"
                  className="h-8"
                  disabled={isDisabled}
                  onClick={handleClearImages}
                >
                  クリア
                </Button>
              )}
            </div>

            {isEditMode && (
              <input
                ref={imageInputRef}
                type="file"
                accept="image/*"
                multiple
                style={{ display: "none" }}
                onChange={handleSelectImages}
              />
            )}

            <div className="rounded-xl border border-slate-300 bg-slate-50 p-4">
              {!hasImages && isEditMode && (
                <div
                  className="flex cursor-pointer flex-col items-center justify-center rounded-xl border border-dashed border-slate-300 bg-white px-6 py-10 text-center transition hover:bg-slate-50"
                  onClick={openPicker}
                  onDrop={handleDropImages}
                  onDragOver={handleDragOverImages}
                  role="button"
                  tabIndex={0}
                  title="クリックで画像を追加"
                >
                  <div className="mb-3 text-slate-400">
                    <ImageIcon />
                  </div>
                  <div className="text-sm font-semibold text-slate-800">
                    画像を追加
                  </div>
                  <div className="mt-1 text-xs text-slate-500">
                    クリックで選択（複数可） / ドロップでも追加できます
                  </div>
                </div>
              )}

              {!hasImages && isViewMode && (
                <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-slate-300 bg-white px-6 py-10 text-center">
                  <div className="mb-3 text-slate-400">
                    <ImageIcon />
                  </div>
                  <div className="text-sm font-semibold text-slate-800">
                    画像はありません
                  </div>
                </div>
              )}

              {hasImages && (
                <div className="space-y-3">
                  <div
                    className="relative overflow-visible"
                    onDrop={isEditMode ? handleDropImages : undefined}
                    onDragOver={isEditMode ? handleDragOverImages : undefined}
                    title={isEditMode ? "クリックで画像追加" : undefined}
                  >
                    <div
                      className={[
                        "flex items-center justify-center overflow-hidden rounded-xl border border-slate-200 bg-white",
                        isEditMode ? "cursor-pointer" : "",
                      ].join(" ")}
                      style={{ minHeight: 260 }}
                      onClick={openPicker}
                      role={isEditMode ? "button" : undefined}
                      tabIndex={isEditMode ? 0 : undefined}
                    >
                      {mainImage && (
                        <img
                          src={mainImage.url}
                          alt={mainImage.name}
                          className="max-h-[360px] w-full object-contain"
                        />
                      )}
                    </div>

                    {isEditMode && (
                      <DeleteButton
                        size="md"
                        disabled={isDisabled}
                        ariaLabel="remove main image"
                        onClick={(event) => {
                          event.stopPropagation();
                          handleRemoveImageAt(mainImageIndex);
                        }}
                      />
                    )}

                    <div className="mt-2 flex items-center justify-between text-xs text-slate-500">
                      <div>
                        {isEditMode
                          ? `${previewImages.length} 枚（×で削除 / クリックで追加）`
                          : `${previewImages.length} 枚`}
                      </div>
                    </div>
                  </div>

                  <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
                    {thumbIndices.map((index) => {
                      const item = previewImages[index];
                      if (!item) return null;

                      return (
                        <div
                          key={item.key}
                          className={[
                            "relative overflow-visible rounded-xl border border-slate-200 bg-white",
                            isEditMode ? "cursor-pointer" : "",
                          ].join(" ")}
                          onClick={() => handleSelectMainImage(index)}
                          role={isEditMode ? "button" : undefined}
                          tabIndex={isEditMode ? 0 : undefined}
                          title={
                            isEditMode
                              ? "クリックでメインに設定"
                              : undefined
                          }
                        >
                          <div className="aspect-square overflow-hidden rounded-xl bg-slate-100">
                            <img
                              src={item.url}
                              alt={item.name}
                              className="h-full w-full object-cover"
                            />
                          </div>

                          {isEditMode && (
                            <DeleteButton
                              size="sm"
                              disabled={isDisabled}
                              ariaLabel="remove image"
                              onClick={(event) => {
                                event.stopPropagation();
                                handleRemoveImageAt(index);
                              }}
                            />
                          )}
                        </div>
                      );
                    })}

                    {isEditMode && (
                      <div
                        className="flex aspect-square cursor-pointer flex-col items-center justify-center rounded-xl border border-dashed border-slate-300 bg-white text-slate-500 transition hover:bg-slate-50"
                        onClick={openPicker}
                        onDrop={handleDropImages}
                        onDragOver={handleDragOverImages}
                        role="button"
                        tabIndex={0}
                        title="クリックで画像を追加"
                      >
                        <div className="mb-1">
                          <PlusIcon />
                        </div>
                        <div className="text-xs font-medium">
                          画像を追加
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              )}
            </div>
          </div>

          <div className="space-y-2">
            <label
              htmlFor="sales-input-title"
              className="text-sm font-medium text-slate-700"
            >
              タイトル
            </label>

            {isEditMode ? (
              <input
                id="sales-input-title"
                type="text"
                value={inputTitle}
                onChange={(event) => setInputTitle(event.target.value)}
                placeholder="タイトルを入力してください"
                disabled={isDisabled}
                className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none transition focus:border-slate-400 focus:ring-2 focus:ring-slate-200 disabled:cursor-not-allowed disabled:bg-slate-50"
              />
            ) : (
              <div className="min-h-10 w-full rounded-md border border-slate-200 bg-slate-50 px-3 py-2 text-sm text-slate-900">
                {formatViewText(inputTitle)}
              </div>
            )}
          </div>

          <div className="space-y-2">
            <label
              htmlFor="sales-input-text"
              className="text-sm font-medium text-slate-700"
            >
              文章
            </label>

            {isEditMode ? (
              <textarea
                id="sales-input-text"
                value={text}
                onChange={(event) => setText(event.target.value)}
                placeholder="文章を入力してください"
                disabled={isDisabled}
                className="min-h-[140px] w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none transition focus:border-slate-400 focus:ring-2 focus:ring-slate-200 disabled:cursor-not-allowed disabled:bg-slate-50"
              />
            ) : (
              <div className="min-h-[140px] w-full whitespace-pre-wrap rounded-md border border-slate-200 bg-slate-50 px-3 py-2 text-sm leading-6 text-slate-900">
                {formatViewText(text)}
              </div>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}