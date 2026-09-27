// frontend/mall/src/features/howToUse/presentation/components/common/HowToUseVideo.tsx

import { useEffect, useRef, useState } from "react";

import Preview from "../../../../../components/ui/Preview";
import TextState from "../../../../../components/ui/TextState";
import { getHowToUseVideoUrl } from "../../../infrastructure/howToUseVideoUrl";

type HowToUseVideoProps = {
  storagePath: string;
  label: string;
  variant?: "default" | "iphone-12-pro";
};

export default function HowToUseVideo({
  storagePath,
  label,
  variant = "default",
}: HowToUseVideoProps) {
  const src = getHowToUseVideoUrl(storagePath);
  const videoRef = useRef<HTMLVideoElement | null>(null);
  const [loading, setLoading] = useState(true);
  const [previewOpen, setPreviewOpen] = useState(false);

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting && !previewOpen) {
          void video.play().catch(() => undefined);
          return;
        }

        video.pause();
      },
      {
        threshold: 0.35,
      },
    );

    observer.observe(video);

    return () => {
      observer.disconnect();
      video.pause();
    };
  }, [src, previewOpen]);

  useEffect(() => {
    if (!previewOpen) return;
    videoRef.current?.pause();
  }, [previewOpen]);

  const figureClassName = [
    "how-to-use-figure",
    variant === "iphone-12-pro" ? "how-to-use-figure--iphone-12-pro" : "",
  ].filter(Boolean).join(" ");

  const imageWrapClassName = [
    "how-to-use-figure__image-wrap",
    variant === "iphone-12-pro" ? "how-to-use-figure__image-wrap--iphone-12-pro" : "",
  ].filter(Boolean).join(" ");

  return (
    <>
      <figure className={figureClassName}>
        <div className={imageWrapClassName} aria-busy={loading}>
          {loading ? (
            <div className="how-to-use-figure__loading" aria-live="polite">
              <TextState variant="loading">読み込み中...</TextState>
            </div>
          ) : null}

          <video
            ref={videoRef}
            src={src}
            aria-label={label}
            className={[
              "how-to-use-figure__video",
              loading ? "how-to-use-figure__video--loading" : "",
            ].filter(Boolean).join(" ")}
            muted
            loop
            playsInline
            preload="auto"
            onCanPlay={() => {
              setLoading(false);
            }}
            onError={() => {
              setLoading(false);
            }}
            onClick={() => {
              if (!loading) {
                setPreviewOpen(true);
              }
            }}
          />
        </div>
      </figure>

      <Preview
        open={previewOpen}
        src={src}
        alt={label}
        type="video"
        onClose={() => {
          setPreviewOpen(false);
        }}
      />
    </>
  );
}