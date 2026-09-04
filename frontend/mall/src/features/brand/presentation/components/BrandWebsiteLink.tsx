// frontend/amol/src/features/brand/presentation/components/BrandWebsiteLink.tsx

type BrandWebsiteLinkProps = {
  url: string;
};

export default function BrandWebsiteLink({
  url,
}: BrandWebsiteLinkProps) {
  return (
    <a
      className="brand-page-link"
      href={url}
      target="_blank"
      rel="noopener noreferrer"
    >
      公式サイトを見る
    </a>
  );
}