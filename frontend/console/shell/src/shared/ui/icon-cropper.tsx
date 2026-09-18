// frontend/console/shell/src/shared/ui/icon-cropper.tsx

import { useCallback, useEffect, useMemo, useRef, useState, type CSSProperties, type KeyboardEvent, type PointerEvent as ReactPointerEvent, type SyntheticEvent } from "react";

import type { IconCropPosition } from "../types/iconCrop";

export type IconCropperProps = {
  src: string;
  position: IconCropPosition;
  scale: number;
  onPositionChange: (position: IconCropPosition) => void;
  onScaleChange: (scale: number) => void;
  onViewportSizeChange?: (size: number) => void;
  alt?: string;
  disabled?: boolean;
  minScale?: number;
  maxScale?: number;
};

type ImageSize = {
  width: number;
  height: number;
};

type DragState = {
  pointerId: number;
  startClientX: number;
  startClientY: number;
  startPosition: IconCropPosition;
};

const DEFAULT_MIN_SCALE = 1;
const DEFAULT_MAX_SCALE = 3;
const SCALE_STEP = 0.01;
const POSITION_EPSILON = 0.01;

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}

function positionsEqual(left: IconCropPosition, right: IconCropPosition): boolean {
  return Math.abs(left.x - right.x) < POSITION_EPSILON && Math.abs(left.y - right.y) < POSITION_EPSILON;
}

function clampPosition({
  position,
  imageSize,
  viewportSize,
  scale,
}: {
  position: IconCropPosition;
  imageSize: ImageSize | null;
  viewportSize: number;
  scale: number;
}): IconCropPosition {
  if (!imageSize || imageSize.width <= 0 || imageSize.height <= 0 || viewportSize <= 0) {
    return position;
  }

  const baseScale = Math.max(viewportSize / imageSize.width, viewportSize / imageSize.height);
  const renderedWidth = imageSize.width * baseScale * scale;
  const renderedHeight = imageSize.height * baseScale * scale;
  const maxX = Math.max(0, (renderedWidth - viewportSize) / 2);
  const maxY = Math.max(0, (renderedHeight - viewportSize) / 2);

  return {
    x: clamp(position.x, -maxX, maxX),
    y: clamp(position.y, -maxY, maxY),
  };
}

export function IconCropper({
  src,
  position,
  scale,
  onPositionChange,
  onScaleChange,
  onViewportSizeChange,
  alt = "アイコン画像の切り抜きプレビュー",
  disabled = false,
  minScale = DEFAULT_MIN_SCALE,
  maxScale = DEFAULT_MAX_SCALE,
}: IconCropperProps) {
  const viewportRef = useRef<HTMLDivElement | null>(null);
  const dragStateRef = useRef<DragState | null>(null);
  const [viewportSize, setViewportSize] = useState(0);
  const [imageSize, setImageSize] = useState<ImageSize | null>(null);
  const [dragging, setDragging] = useState(false);

  const normalizedMinScale = Math.max(1, minScale);
  const normalizedMaxScale = Math.max(normalizedMinScale, maxScale);
  const normalizedScale = clamp(scale, normalizedMinScale, normalizedMaxScale);

  const updateViewportSize = useCallback(() => {
    const viewport = viewportRef.current;
    if (!viewport) return;

    const rect = viewport.getBoundingClientRect();
    const nextSize = Math.min(rect.width, rect.height);
    if (nextSize <= 0) return;

    setViewportSize((currentSize) => {
      if (Math.abs(currentSize - nextSize) < 0.5) return currentSize;
      return nextSize;
    });
  }, []);

  useEffect(() => {
    updateViewportSize();

    const viewport = viewportRef.current;
    if (!viewport) return;

    if (typeof ResizeObserver === "undefined") {
      window.addEventListener("resize", updateViewportSize);
      return () => window.removeEventListener("resize", updateViewportSize);
    }

    const observer = new ResizeObserver(updateViewportSize);
    observer.observe(viewport);

    return () => observer.disconnect();
  }, [updateViewportSize]);

  useEffect(() => {
    if (viewportSize > 0) {
      onViewportSizeChange?.(viewportSize);
    }
  }, [onViewportSizeChange, viewportSize]);

  useEffect(() => {
    const nextPosition = clampPosition({
      position,
      imageSize,
      viewportSize,
      scale: normalizedScale,
    });

    if (!positionsEqual(nextPosition, position)) {
      onPositionChange(nextPosition);
    }
  }, [imageSize, normalizedScale, onPositionChange, position, viewportSize]);

  useEffect(() => {
    if (scale !== normalizedScale) {
      onScaleChange(normalizedScale);
    }
  }, [normalizedScale, onScaleChange, scale]);

  useEffect(() => {
    dragStateRef.current = null;
    setDragging(false);
    setImageSize(null);
  }, [src]);

  const baseImageSize = useMemo(() => {
    if (!imageSize || viewportSize <= 0 || imageSize.width <= 0 || imageSize.height <= 0) {
      return null;
    }

    const baseScale = Math.max(viewportSize / imageSize.width, viewportSize / imageSize.height);

    return {
      width: imageSize.width * baseScale,
      height: imageSize.height * baseScale,
    };
  }, [imageSize, viewportSize]);

  const imageStyle = useMemo<CSSProperties>(() => {
    if (!baseImageSize) {
      return { visibility: "hidden" };
    }

    return {
      width: `${baseImageSize.width}px`,
      height: `${baseImageSize.height}px`,
      transform: `translate(-50%, -50%) translate3d(${position.x}px, ${position.y}px, 0) scale(${normalizedScale})`,
      transformOrigin: "center center",
    };
  }, [baseImageSize, normalizedScale, position.x, position.y]);

  const moveTo = useCallback(
    (nextPosition: IconCropPosition) => {
      const clampedPosition = clampPosition({
        position: nextPosition,
        imageSize,
        viewportSize,
        scale: normalizedScale,
      });

      if (!positionsEqual(clampedPosition, position)) {
        onPositionChange(clampedPosition);
      }
    },
    [imageSize, normalizedScale, onPositionChange, position, viewportSize],
  );

  const handleImageLoad = (event: SyntheticEvent<HTMLImageElement>) => {
    const image = event.currentTarget;

    setImageSize({
      width: image.naturalWidth,
      height: image.naturalHeight,
    });
  };

  const handlePointerDown = (event: ReactPointerEvent<HTMLDivElement>) => {
    if (disabled || !imageSize) return;

    event.preventDefault();
    event.currentTarget.setPointerCapture(event.pointerId);

    dragStateRef.current = {
      pointerId: event.pointerId,
      startClientX: event.clientX,
      startClientY: event.clientY,
      startPosition: position,
    };

    setDragging(true);
  };

  const handlePointerMove = (event: ReactPointerEvent<HTMLDivElement>) => {
    const dragState = dragStateRef.current;

    if (disabled || !dragState || dragState.pointerId !== event.pointerId) {
      return;
    }

    event.preventDefault();

    moveTo({
      x: dragState.startPosition.x + (event.clientX - dragState.startClientX),
      y: dragState.startPosition.y + (event.clientY - dragState.startClientY),
    });
  };

  const finishPointerDrag = (event: ReactPointerEvent<HTMLDivElement>) => {
    const dragState = dragStateRef.current;

    if (!dragState || dragState.pointerId !== event.pointerId) {
      return;
    }

    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId);
    }

    dragStateRef.current = null;
    setDragging(false);
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (disabled || !imageSize) return;

    const distance = event.shiftKey ? 12 : 4;

    switch (event.key) {
      case "ArrowLeft":
        event.preventDefault();
        moveTo({ x: position.x - distance, y: position.y });
        break;
      case "ArrowRight":
        event.preventDefault();
        moveTo({ x: position.x + distance, y: position.y });
        break;
      case "ArrowUp":
        event.preventDefault();
        moveTo({ x: position.x, y: position.y - distance });
        break;
      case "ArrowDown":
        event.preventDefault();
        moveTo({ x: position.x, y: position.y + distance });
        break;
      default:
        break;
    }
  };

  const handleScaleChange = (nextScaleRaw: number) => {
    if (disabled) return;

    const nextScale = clamp(nextScaleRaw, normalizedMinScale, normalizedMaxScale);
    onScaleChange(nextScale);

    const nextPosition = clampPosition({
      position,
      imageSize,
      viewportSize,
      scale: nextScale,
    });

    if (!positionsEqual(nextPosition, position)) {
      onPositionChange(nextPosition);
    }
  };

  return (
    <div className="icon-cropper">
      <div
        ref={viewportRef}
        className={[
          "icon-cropper__viewport",
          dragging ? "icon-cropper__viewport--dragging" : "",
          disabled ? "icon-cropper__viewport--disabled" : "",
        ].filter(Boolean).join(" ")}
        role="application"
        tabIndex={disabled ? -1 : 0}
        aria-label="アイコン画像の表示位置を調整"
        onPointerDown={handlePointerDown}
        onPointerMove={handlePointerMove}
        onPointerUp={finishPointerDrag}
        onPointerCancel={finishPointerDrag}
        onKeyDown={handleKeyDown}
      >
        <img
          src={src}
          alt={alt}
          className="icon-cropper__image"
          style={imageStyle}
          draggable={false}
          onLoad={handleImageLoad}
          onDragStart={(event) => event.preventDefault()}
        />
        <div className="icon-cropper__frame" aria-hidden="true" />
      </div>

      <p className="icon-cropper__help">画像をドラッグして表示位置を調整してください。</p>

      <label className="icon-cropper__zoom">
        <span className="icon-cropper__zoom-label">拡大</span>
        <input
          type="range"
          min={normalizedMinScale}
          max={normalizedMaxScale}
          step={SCALE_STEP}
          value={normalizedScale}
          disabled={disabled}
          className="icon-cropper__zoom-input"
          aria-label="アイコン画像の拡大率"
          onChange={(event) => handleScaleChange(Number(event.target.value))}
        />
      </label>
    </div>
  );
}

export default IconCropper;