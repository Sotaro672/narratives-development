// frontend\amol\src\features\shared\presentation\components\ProductIdentity.tsx

export type ProductIdentityProps = {
  brandName?: string | null;
  productName?: string | null;
  tokenName?: string | null;
};

export default function ProductIdentity({
  brandName,
  productName,
  tokenName,
}: ProductIdentityProps) {
  const safeBrandName = brandName?.trim() || "ブランド名未設定";
  const safeProductName = productName?.trim() || tokenName?.trim() || "商品名未設定";

  return (
    <>
      <p className="product-detail__brand">{safeBrandName}</p>
      <h1 className="product-detail__title">{safeProductName}</h1>
    </>
  );
}