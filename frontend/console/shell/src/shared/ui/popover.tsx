// frontend/console/shell/src/shared/ui/popover.tsx

import React, {
  createContext,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";
import type {
  CSSProperties,
  ReactNode,
} from "react";

import "./popover.css";

interface PopoverCtx {
  open: boolean;
  setOpen: (v: boolean) => void;
  triggerRef: React.MutableRefObject<HTMLElement | null>;
  contentRef: React.MutableRefObject<HTMLDivElement | null>;
}

const PopoverContext =
  createContext<PopoverCtx | null>(null);

function usePopover() {
  const ctx = useContext(PopoverContext);

  if (!ctx) {
    throw new Error(
      "Popover components must be used within <Popover>",
    );
  }

  return ctx;
}

export function Popover({
  children,
}: {
  children: ReactNode;
}) {
  const [open, setOpen] = useState(false);
  const triggerRef =
    useRef<HTMLElement | null>(null);
  const contentRef =
    useRef<HTMLDivElement | null>(null);

  // 外側クリック・ESCで閉じる
  useEffect(() => {
    const onMouseDown = (
      event: MouseEvent,
    ) => {
      const target =
        event.target as Node;

      if (
        open &&
        contentRef.current &&
        !contentRef.current.contains(target) &&
        triggerRef.current &&
        !triggerRef.current.contains(target)
      ) {
        setOpen(false);
      }
    };

    const onKeyDown = (
      event: KeyboardEvent,
    ) => {
      if (event.key === "Escape") {
        setOpen(false);
      }
    };

    document.addEventListener(
      "mousedown",
      onMouseDown,
    );
    document.addEventListener(
      "keydown",
      onKeyDown,
    );

    return () => {
      document.removeEventListener(
        "mousedown",
        onMouseDown,
      );
      document.removeEventListener(
        "keydown",
        onKeyDown,
      );
    };
  }, [open]);

  return (
    <PopoverContext.Provider
      value={{
        open,
        setOpen,
        triggerRef,
        contentRef,
      }}
    >
      {children}
    </PopoverContext.Provider>
  );
}

export function PopoverTrigger({
  children,
}: {
  children: ReactNode;
}) {
  const {
    open,
    setOpen,
    triggerRef,
  } = usePopover();

  return (
    <span
      ref={(element) => {
        triggerRef.current = element;
      }}
      className="popover__trigger"
      onClick={(event) => {
        event.stopPropagation();
        setOpen(!open);
      }}
      aria-haspopup="dialog"
      aria-expanded={open}
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
  const {
    open,
    triggerRef,
    contentRef,
  } = usePopover();

  const [pos, setPos] = useState<{
    top: number;
    left: number;
    width: number;
  }>({
    top: 0,
    left: 0,
    width: 0,
  });

  // 座標をビューポート基準で算出する
  const recalc = () => {
    const rect =
      triggerRef.current?.getBoundingClientRect();

    if (!rect) {
      return;
    }

    let left = rect.left;

    if (align === "center") {
      left =
        rect.left +
        rect.width / 2;
    }

    if (align === "end") {
      left = rect.right;
    }

    setPos({
      top: rect.bottom + offset,
      left,
      width: rect.width,
    });
  };

  useEffect(() => {
    if (!open) {
      return;
    }

    recalc();

    const onScroll = () => {
      recalc();
    };

    const onResize = () => {
      recalc();
    };

    window.addEventListener(
      "scroll",
      onScroll,
      { passive: true },
    );
    window.addEventListener(
      "resize",
      onResize,
      { passive: true },
    );

    // overflow: auto / scrollを持つ祖先のスクロールも捕捉する
    const parents: Element[] = [];
    let element: Element | null =
      triggerRef.current;

    while (
      element &&
      element.parentElement
    ) {
      element =
        element.parentElement;

      const computedStyle =
        getComputedStyle(element);

      if (
        /(auto|scroll)/.test(
          computedStyle.overflow +
          computedStyle.overflowY +
          computedStyle.overflowX,
        )
      ) {
        parents.push(element);

        element.addEventListener(
          "scroll",
          onScroll,
          { passive: true },
        );
      }
    }

    return () => {
      window.removeEventListener(
        "scroll",
        onScroll,
      );
      window.removeEventListener(
        "resize",
        onResize,
      );

      parents.forEach((parent) => {
        parent.removeEventListener(
          "scroll",
          onScroll,
        );
      });
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, align, offset]);

  if (!open) {
    return null;
  }

  return (
    <div
      ref={contentRef}
      role="dialog"
      className={[
        "popover__content",
        className,
      ]
        .filter(Boolean)
        .join(" ")}
      style={{
        top: pos.top,
        left: pos.left,
        transform:
          align === "center"
            ? "translateX(-50%)"
            : align === "end"
              ? "translateX(-100%)"
              : undefined,
        minWidth: Math.max(
          220,
          pos.width + 12,
        ),
        ...style,
      }}
    >
      {children}
    </div>
  );
}