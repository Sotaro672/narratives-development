// frontend/mall/src/features/howToUse/presentation/components/common/HowToUseFigure.tsx

type HowToUseFigureProps = {
  src: string;
  alt: string;
  caption?: string;
};

export default function HowToUseFigure({
  src,
  alt,
  caption,
}: HowToUseFigureProps) {
  return (
    <figure className="how-to-use-figure">
      <div className="how-to-use-figure__image-wrap">
        <img
          src={src}
          alt={alt}
          className="how-to-use-figure__image"
          loading="lazy"
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