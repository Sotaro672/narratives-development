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
  const isIPhone12Pro = variant === "iphone-12-pro";

  return (
    <figure
      className="how-to-use-figure"
      style={
        isIPhone12Pro
          ? {
              maxWidth: "390px",
              marginLeft: "auto",
              marginRight: "auto",
            }
          : undefined
      }
    >
      <div
        className="how-to-use-figure__image-wrap"
        style={
          isIPhone12Pro
            ? {
                aspectRatio: "390 / 844",
              }
            : undefined
        }
      >
        <video
          src={src}
          aria-label={label}
          className="how-to-use-figure__video"
          autoPlay
          muted
          loop
          playsInline
          preload="metadata"
          style={
            isIPhone12Pro
              ? {
                  width: "100%",
                  height: "100%",
                  objectFit: "contain",
                }
              : undefined
          }
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