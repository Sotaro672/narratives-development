// frontend\amol\src\features\shared\presentation\components\ProductIdentity.tsx
export type ResaleProductIdentityProps = {
  brandName?: string | null;
  productName?: string | null;
  tokenName?: string | null;
};

export default function ResaleProductIdentity({
  brandName,
  productName,
  tokenName,
}: ResaleProductIdentityProps) {
  const safeBrandName = brandName?.trim() || "ブランド名未設定";
  const safeProductName =
    productName?.trim() ||
    tokenName?.trim() ||
    "商品名未設定";

  return (
    <>
      <p className="resale-product-detail__brand">
        {safeBrandName}
      </p>

      <h1 className="resale-product-detail__title">
        {safeProductName}
      </h1>
    </>
  );
}