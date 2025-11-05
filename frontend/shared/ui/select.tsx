// frontend/shared/ui/select.tsx
"use client";

import * as React from "react";

/* ------------------------------- 内部ユーティリティ ------------------------------- */

/** className 結合ヘルパー */
function cn(...classes: Array<string | undefined | false | null>) {
  return classes.filter(Boolean).join(" ");
}

/* ------------------------------- 簡易アイコン定義 ------------------------------- */

function CheckIcon(props: React.SVGProps<SVGSVGElement>) {
  return (
    <svg
      {...props}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <polyline points="20 6 9 17 4 12" />
    </svg>
  );
}

function ChevronDownIcon(props: React.SVGProps<SVGSVGElement>) {
  return (
    <svg
      {...props}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <polyline points="6 9 12 15 18 9" />
    </svg>
  );
}

function ChevronUpIcon(props: React.SVGProps<SVGSVGElement>) {
  return (
    <svg
      {...props}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <polyline points="18 15 12 9 6 15" />
    </svg>
  );
}

/* ------------------------------- Select コンポーネント群 ------------------------------- */

/**
 * Select 基本構成
 * - <Select> → コンテナ
 * - <SelectTrigger> → トリガーボタン
 * - <SelectContent> → 選択肢リスト
 * - <SelectItem> → 各選択肢
 * - <SelectValue> → 選択中の値表示
 * - <SelectLabel> / <SelectSeparator> などもオプション
 */

interface SelectOption {
  value: string;
  label: string;
}

interface SelectProps {
  options: SelectOption[];
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
  label?: string;
  className?: string;
  size?: "sm" | "default";
}

/** メイン Select */
export function Select({
  options,
  value,
  onChange,
  placeholder = "選択してください",
  label,
  className,
  size = "default",
}: SelectProps) {
  const [open, setOpen] = React.useState(false);

  return (
    <div className={cn("relative inline-block w-full", className)}>
      {label && (
        <div className="mb-1 text-xs text-gray-500">{label}</div>
      )}

      {/* Trigger */}
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className={cn(
          "flex w-full items-center justify-between gap-2 rounded-md border border-gray-300 bg-white px-3 text-sm transition-all",
          size === "default" ? "h-9 py-2" : "h-8 py-1.5",
          "hover:bg-gray-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-400"
        )}
      >
        <span className="truncate text-gray-800">
          {value || placeholder}
        </span>
        <ChevronDownIcon className="w-4 h-4 opacity-70" />
      </button>

      {/* Content */}
      {open && (
        <div
          className="absolute z-10 mt-1 w-full rounded-md border border-gray-200 bg-white shadow-lg"
          role="listbox"
        >
          {options.map((opt) => (
            <div
              key={opt.value}
              className={cn(
                "flex cursor-pointer items-center justify-between px-3 py-2 text-sm hover:bg-blue-50",
                value === opt.value && "bg-blue-100 text-blue-700 font-medium"
              )}
              onClick={() => {
                onChange(opt.value);
                setOpen(false);
              }}
            >
              <span>{opt.label}</span>
              {value === opt.value && <CheckIcon className="w-4 h-4" />}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

/* -----------------------------------------------------------
   シンプルな共通 API に統一した Select 実装
   - 完全独立（@radix-ui, lucide-react 非依存）
   - Tailwindベースの外観
   - props: { options, value, onChange, placeholder, label }
   ----------------------------------------------------------- */
