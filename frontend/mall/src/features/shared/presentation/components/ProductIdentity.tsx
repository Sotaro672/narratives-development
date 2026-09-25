// frontend/mall/src/features/shared/presentation/components/ProductIdentity.tsx

export type ProductIdentityProps = {
  brandName?: string | null;
  productName?: string | null;
  tokenName?: string | null;
};

export default function ProductIdentity({
  productName,
  tokenName,
}: ProductIdentityProps) {
  const safeProductName =
    productName?.trim() || tokenName?.trim() || "商品名未設定";

  return (
    <h1 className="product-detail__title">
      {safeProductName}
    </h1>
  );
}