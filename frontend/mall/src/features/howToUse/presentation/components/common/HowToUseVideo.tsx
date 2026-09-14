// frontend/mall/src/features/howToUse/presentation/components/common/HowToUseVideo.tsx

type HowToUseVideoProps = {
  src: string;
  label: string;
  caption?: string;
  variant?: "default" | "iphone-12-pro";
};

export default function HowToUseVideo({
  src,
  label,
  caption,
  variant = "default",
}: HowToUseVideoProps) {
  const figureClassName = [
    "how-to-use-figure",
    variant === "iphone-12-pro"
      ? "how-to-use-figure--iphone-12-pro"
      : "",
  ]
    .filter(Boolean)
    .join(" ");

  const imageWrapClassName = [
    "how-to-use-figure__image-wrap",
    variant === "iphone-12-pro"
      ? "how-to-use-figure__image-wrap--iphone-12-pro"
      : "",
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <figure className={figureClassName}>
      <div className={imageWrapClassName}>
        <video
          src={src}
          aria-label={label}
          className="how-to-use-figure__video"
          autoPlay
          muted
          loop
          playsInline
          preload="metadata"
        />
      </div>

      {caption ? (
        <figcaption className="how-to-use-figure__caption">
          {caption}
        </figcaption>
      ) : null}
    </figure>
  );
}