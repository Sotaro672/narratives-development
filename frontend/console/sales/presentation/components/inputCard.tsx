// frontend/console/sales/src/presentation/components/inputCard.tsx
import { useEffect, useMemo, useRef, useState } from "react";
import type * as React from "react";
import { Button } from "../../../shell/src/shared/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../shell/src/shared/ui/card";

type SubmitPayload = {
  title: string;
  text: string;
  images: File[];
};

type Props = {
  title?: string;
  initialTitle?: string;
  initialText?: string;
  initialImages?: File[];
  saving?: boolean;
  sending?: boolean;
  onSave?: (payload: SubmitPayload) => void | Promise<void>;
  onSend?: (payload: SubmitPayload) => void | Promise<void>;
};

type PreviewImage = {
  key: string;
  file: File;
  url: string;
};

function fileKey(file: File, index: number): string {
  return `${file.name}-${file.size}-${file.lastModified}-${index}`;
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

function getImageIdentity(file: File): string {
  return `${file.name}-${file.size}-${file.lastModified}`;
}

export default function InputCard({
  title = "入力",
  initialTitle = "",
  initialText = "",
  initialImages = [],
  saving = false,
  sending = false,
  onSave,
  onSend,
}: Props) {
  const [inputTitle, setInputTitle] = useState(initialTitle);
  const [text, setText] = useState(initialText);
  const [images, setImages] = useState<File[]>(initialImages);
  const [mainImageIndex, setMainImageIndex] = useState(0);

  const imageInputRef = useRef<HTMLInputElement | null>(null);

  useEffect(() => {
    setInputTitle(initialTitle);
  }, [initialTitle]);

  useEffect(() => {
    setText(initialText);
  }, [initialText]);

  useEffect(() => {
    setImages(initialImages);
    setMainImageIndex(0);
  }, [initialImages]);

  const previewImages = useMemo<PreviewImage[]>(() => {
    return images.map((file, index) => ({
      key: fileKey(file, index),
      file,
      url: URL.createObjectURL(file),
    }));
  }, [images]);

  useEffect(() => {
    return () => {
      previewImages.forEach((item) => URL.revokeObjectURL(item.url));
    };
  }, [previewImages]);

  useEffect(() => {
    if (images.length === 0) {
      if (mainImageIndex !== 0) setMainImageIndex(0);
      return;
    }
    if (mainImageIndex > images.length - 1) {
      setMainImageIndex(images.length - 1);
    }
  }, [images, mainImageIndex]);

  const isBusy = saving || sending;
  const hasImages = previewImages.length > 0;
  const mainImage = previewImages[mainImageIndex] ?? null;
  const thumbIndices = previewImages
    .map((_, index) => index)
    .filter((index) => index !== mainImageIndex);

  const openPicker = () => {
    if (isBusy) return;
    imageInputRef.current?.click();
  };

  const addImages = (nextFiles: File[]) => {
    if (nextFiles.length === 0) return;

    setImages((prev) => {
      const seen = new Set(prev.map((file) => getImageIdentity(file)));
      const merged = [...prev];

      for (const file of nextFiles) {
        if (!file.type.startsWith("image/")) continue;

        const id = getImageIdentity(file);
        if (seen.has(id)) continue;

        seen.add(id);
        merged.push(file);
      }

      return merged;
    });
  };

  const handleSelectImages = (event: React.ChangeEvent<HTMLInputElement>) => {
    const nextFiles = Array.from(event.target.files ?? []);
    addImages(nextFiles);
    event.target.value = "";
  };

  const handleDropImages = (event: React.DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    if (isBusy) return;

    const nextFiles = Array.from(event.dataTransfer.files ?? []);
    addImages(nextFiles);
  };

  const handleDragOverImages = (event: React.DragEvent<HTMLDivElement>) => {
    event.preventDefault();
  };

  const handleRemoveImageAt = (targetIndex: number) => {
    setImages((prev) => prev.filter((_, index) => index !== targetIndex));

    setMainImageIndex((prev) => {
      if (targetIndex < prev) return prev - 1;
      if (targetIndex === prev) return 0;
      return prev;
    });
  };

  const handleClearImages = () => {
    setImages([]);
    setMainImageIndex(0);
  };

  const handleSave = async () => {
    await onSave?.({
      title: inputTitle.trim(),
      text: text.trim(),
      images,
    });
  };

  const handleSend = async () => {
    await onSend?.({
      title: inputTitle.trim(),
      text: text.trim(),
      images,
    });
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

              {hasImages && (
                <Button
                  type="button"
                  variant="ghost"
                  className="h-8"
                  disabled={isBusy}
                  onClick={handleClearImages}
                >
                  クリア
                </Button>
              )}
            </div>

            <input
              ref={imageInputRef}
              type="file"
              accept="image/*"
              multiple
              style={{ display: "none" }}
              onChange={handleSelectImages}
            />

            <div className="rounded-xl border border-slate-300 bg-slate-50 p-4">
              {!hasImages && (
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

              {hasImages && (
                <div className="space-y-3">
                  <div
                    className="relative"
                    onDrop={handleDropImages}
                    onDragOver={handleDragOverImages}
                    title="クリックで画像追加"
                  >
                    <div
                      className="flex cursor-pointer items-center justify-center overflow-hidden rounded-xl border border-slate-200 bg-white"
                      style={{ minHeight: 260 }}
                      onClick={openPicker}
                      role="button"
                      tabIndex={0}
                    >
                      {mainImage && (
                        <img
                          src={mainImage.url}
                          alt={mainImage.file.name}
                          className="max-h-[360px] w-full object-contain"
                        />
                      )}
                    </div>

                    <button
                      type="button"
                      className="absolute right-2 top-2 flex h-8 w-8 items-center justify-center rounded-full bg-black/60 text-lg text-white disabled:opacity-50"
                      onClick={(event) => {
                        event.stopPropagation();
                        handleRemoveImageAt(mainImageIndex);
                      }}
                      aria-label="remove main image"
                      title="削除"
                      disabled={isBusy}
                    >
                      ×
                    </button>

                    <div className="mt-2 flex items-center justify-between text-xs text-slate-500">
                      <div>{previewImages.length} 枚（×で削除 / クリックで追加）</div>
                    </div>
                  </div>

                  <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
                    {thumbIndices.map((index) => {
                      const item = previewImages[index];
                      if (!item) return null;

                      return (
                        <div
                          key={item.key}
                          className="relative cursor-pointer overflow-hidden rounded-xl border border-slate-200 bg-white"
                          onClick={() => setMainImageIndex(index)}
                          role="button"
                          tabIndex={0}
                          title="クリックでメインに設定"
                        >
                          <div className="aspect-square bg-slate-100">
                            <img
                              src={item.url}
                              alt={item.file.name}
                              className="h-full w-full object-cover"
                            />
                          </div>

                          <button
                            type="button"
                            className="absolute right-2 top-2 flex h-7 w-7 items-center justify-center rounded-full bg-black/60 text-sm text-white disabled:opacity-50"
                            onClick={(event) => {
                              event.stopPropagation();
                              handleRemoveImageAt(index);
                            }}
                            aria-label="remove image"
                            title="削除"
                            disabled={isBusy}
                          >
                            ×
                          </button>
                        </div>
                      );
                    })}

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
                      <div className="text-xs font-medium">画像を追加</div>
                    </div>
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
            <input
              id="sales-input-title"
              type="text"
              value={inputTitle}
              onChange={(event) => setInputTitle(event.target.value)}
              placeholder="タイトルを入力してください"
              disabled={isBusy}
              className="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none transition focus:border-slate-400 focus:ring-2 focus:ring-slate-200 disabled:cursor-not-allowed disabled:bg-slate-50"
            />
          </div>

          <div className="space-y-2">
            <label
              htmlFor="sales-input-text"
              className="text-sm font-medium text-slate-700"
            >
              文章
            </label>
            <textarea
              id="sales-input-text"
              value={text}
              onChange={(event) => setText(event.target.value)}
              placeholder="文章を入力してください"
              disabled={isBusy}
              className="min-h-[140px] w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none transition focus:border-slate-400 focus:ring-2 focus:ring-slate-200 disabled:cursor-not-allowed disabled:bg-slate-50"
            />
          </div>

          <div className="flex justify-end gap-2 pt-2">
            <Button
              type="button"
              variant="outline"
              disabled={isBusy}
              onClick={handleSave}
            >
              {saving ? "保存中..." : "保存"}
            </Button>
            <Button
              type="button"
              variant="solid"
              disabled={isBusy}
              onClick={handleSend}
            >
              {sending ? "送信中..." : "送信"}
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}