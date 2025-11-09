// frontend/shared/ui/popover.tsx
import React, { createContext, useContext, useEffect, useRef, useState } from "react";
import type { CSSProperties, ReactNode } from "react";

interface PopoverCtx {
  open: boolean;
  setOpen: (v: boolean) => void;
  triggerRef: React.MutableRefObject<HTMLElement | null>;
  contentRef: React.MutableRefObject<HTMLDivElement | null>;
}

const PopoverContext = createContext<PopoverCtx | null>(null);
const usePopover = () => {
  const ctx = useContext(PopoverContext);
  if (!ctx) throw new Error("Popover components must be used within <Popover>");
  return ctx;
};

export function Popover({ children }: { children: ReactNode }) {
  const [open, setOpen] = useState(false);
  const triggerRef = useRef<HTMLElement | null>(null);
  const contentRef = useRef<HTMLDivElement | null>(null);

  // 外側クリック・ESCで閉じる
  useEffect(() => {
    const onMouseDown = (e: MouseEvent) => {
      const t = e.target as Node;
      if (
        open &&
        contentRef.current &&
        !contentRef.current.contains(t) &&
        triggerRef.current &&
        !triggerRef.current.contains(t)
      ) {
        setOpen(false);
      }
    };
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && setOpen(false);
    document.addEventListener("mousedown", onMouseDown);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onMouseDown);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  return (
    <PopoverContext.Provider value={{ open, setOpen, triggerRef, contentRef }}>
      {children}
    </PopoverContext.Provider>
  );
}

export function PopoverTrigger({ children }: { children: ReactNode }) {
  const { open, setOpen, triggerRef } = usePopover();
  return (
    <span
      ref={(el) => {
        triggerRef.current = el;
      }}
      onClick={(e) => {
        e.stopPropagation();
        setOpen(!open);
      }}
      aria-haspopup="dialog"
      aria-expanded={open}
      style={{ display: "inline-flex" }}
    >
      {children}
    </span>
  );
}

export function PopoverContent({
  children,
  align = "start",
  offset = 8,
  style,
  className = "",
}: {
  children: ReactNode;
  align?: "start" | "center" | "end";
  offset?: number;
  style?: CSSProperties;
  className?: string;
}) {
  const { open, triggerRef, contentRef } = usePopover();
  const [pos, setPos] = useState<{ top: number; left: number; width: number }>({
    top: 0,
    left: 0,
    width: 0,
  });

  // 座標をビューポート基準で算出（fixed 用）
  const recalc = () => {
    const rect = triggerRef.current?.getBoundingClientRect();
    if (!rect) return;
    let left = rect.left;
    if (align === "center") left = rect.left + rect.width / 2;
    if (align === "end") left = rect.right;
    setPos({ top: rect.bottom + offset, left, width: rect.width });
  };

  useEffect(() => {
    if (!open) return;
    recalc();
    const onScroll = () => recalc(); // メイン領域スクロール時も再計算
    const onResize = () => recalc();
    window.addEventListener("scroll", onScroll, { passive: true });
    window.addEventListener("resize", onResize, { passive: true });
    // 祖先に overflow: auto な要素がある場合も念のため捕捉
    const parents: Element[] = [];
    let el: Element | null = triggerRef.current || null;
    while (el && el.parentElement) {
      el = el.parentElement;
      if (!el) break;
      const style = getComputedStyle(el);
      if (/(auto|scroll)/.test(style.overflow + style.overflowY + style.overflowX)) {
        parents.push(el);
        el.addEventListener("scroll", onScroll, { passive: true });
      }
    }
    return () => {
      window.removeEventListener("scroll", onScroll);
      window.removeEventListener("resize", onResize);
      parents.forEach((p) => p.removeEventListener("scroll", onScroll));
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, align, offset]);

  if (!open) return null;

  return (
    <div
      ref={contentRef}
      role="dialog"
      className={className}
      style={{
        position: "fixed", // ★ ビューポート基準
        top: pos.top,
        left: align === "start" ? pos.left : undefined,
        transform:
          align === "center"
            ? `translateX(calc(${pos.left}px - 50%))`
            : align === "end"
            ? `translateX(calc(${pos.left}px - 100%))`
            : undefined,
        minWidth: Math.max(220, pos.width + 12),
        background: "#fff",
        border: "1px solid #e5e7eb",
        borderRadius: 10,
        padding: 12,
        boxShadow: "0 8px 24px rgba(15,23,42,0.08)",
        zIndex: 1000,
        ...style,
      }}
    >
      {children}
    </div>
  );
}

