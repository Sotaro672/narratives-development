// frontend/mall/src/features/howToUse/presentation/components/common/HowToUseVideo.tsx

type HowToUseVideoProps = {
  src: string;
  label: string;
  caption?: string;
};

export default function HowToUseVideo({
  src,
  label,
  caption,
}: HowToUseVideoProps) {
  return (
    <figure className="how-to-use-figure">
      <div className="how-to-use-figure__image-wrap">
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