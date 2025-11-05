import { useEffect, useRef, useState } from "react";
import { Filter, Check } from "lucide-react";

type Option = { value: string; label?: string };

interface FilterableTableHeaderProps {
  /** 列名（例: "ブランド"） */
  label: string;
  /** 現在のテーブルに載っている値から作った候補 */
  options: Option[];
  /** 親から渡す選択状態（省略時はローカルで保持） */
  selected?: string[];
  /** 選択変更時に親へ通知（paginate/filter などは親で実施） */
  onChange?: (next: string[]) => void;
  /** ドロップダウンの見出し（省略時は「{label}で絞り込み」） */
  dialogTitle?: string;
}

export default function FilterableTableHeader({
  label,
  options,
  selected,
  onChange,
  dialogTitle,
}: FilterableTableHeaderProps) {
  // 制御/非制御の両対応
  const [internal, setInternal] = useState<string[]>(selected ?? []);
  const isControlled = selected !== undefined;
  const current = isControlled ? selected! : internal;

  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement | null>(null);

  // 外側クリックで閉じる
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (!containerRef.current) return;
      if (!containerRef.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  // ESCで閉じる
  useEffect(() => {
    const handler = (e: KeyboardEvent) => e.key === "Escape" && setOpen(false);
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, []);

  // 親から選択が変わったら同期
  useEffect(() => {
    if (isControlled) setInternal(selected!);
  }, [isControlled, selected]);

  const toggle = (val: string) => {
    const next = current.includes(val)
      ? current.filter((v) => v !== val)
      : [...current, val];

    if (isControlled) onChange?.(next);
    else setInternal(next);
  };

  const clearAll = () => {
    if (isControlled) onChange?.([]);
    else setInternal([]);
  };

  const apply = () => {
    // 非制御の場合のみ、外へ最終状態を通知
    if (!isControlled) onChange?.(internal);
    setOpen(false);
  };

  return (
    <div className="fth-wrap" ref={containerRef}>
      <span>{label}</span>
      <button
        className="lp-th-filter"
        aria-haspopup="dialog"
        aria-expanded={open}
        onClick={() => setOpen((v) => !v)}
        title={`${label}で絞り込む`}
      >
        <Filter size={16} />
      </button>

      {open && (
        <div
          className="fth-popover"
          role="dialog"
          aria-label={`${label} のフィルター`}
        >
          <div className="fth-header">
            {dialogTitle ?? `${label}で絞り込み`}
          </div>

          <div className="fth-list" role="group" aria-label="候補">
            {options.length === 0 && (
              <div className="fth-empty">候補がありません</div>
            )}

            {options.map((opt) => {
              const checked = current.includes(opt.value);
              return (
                <label key={opt.value} className={`fth-item ${checked ? "is-checked" : ""}`}>
                  <input
                    type="checkbox"
                    checked={checked}
                    onChange={() => toggle(opt.value)}
                  />
                  <span className="fth-check">
                    <Check size={14} aria-hidden />
                  </span>
                  <span className="fth-label">{opt.label ?? opt.value}</span>
                </label>
              );
            })}
          </div>

          <div className="fth-actions">
            <button className="lp-btn" onClick={clearAll}>クリア</button>
            <button className="lp-btn lp-btn-primary" onClick={apply}>適用</button>
          </div>
        </div>
      )}
    </div>
  );
}
