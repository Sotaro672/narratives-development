// frontend/amol/src/features/brand/presentation/components/BrandWebsiteLink.tsx

type BrandWebsiteLinkProps = {
  url: string;
};

function normalizeWebsiteUrl(
  sourceUrl: string,
): string {
  const normalizedUrl =
    sourceUrl.trim();

  if (!normalizedUrl) {
    return "";
  }

  if (
    normalizedUrl.startsWith(
      "http://",
    ) ||
    normalizedUrl.startsWith(
      "https://",
    )
  ) {
    return normalizedUrl;
  }

  return `https://${normalizedUrl}`;
}

export default function BrandWebsiteLink({
  url,
}: BrandWebsiteLinkProps) {
  const href =
    normalizeWebsiteUrl(url);

  if (!href) {
    return null;
  }

  return (
    <a
      className="brand-page-link"
      href={href}
      target="_blank"
      rel="noopener noreferrer"
    >
      公式サイトを見る
    </a>
  );
}