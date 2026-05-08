//frontend\amol\src\features\catalog\components\TokenInfoCard.tsx
import type { CatalogTokenBlueprint } from "../types";

type TokenInfoCardProps = {
  tokenBlueprint: CatalogTokenBlueprint;
};

export default function TokenInfoCard({
  tokenBlueprint,
}: TokenInfoCardProps) {
  return (
    <section className="catalog-page-card">
      <h2 className="catalog-page-card-title">トークン情報</h2>

      <div className="catalog-page-token">
        {tokenBlueprint.tokenIcon ? (
          <img
            src={tokenBlueprint.tokenIcon}
            alt={tokenBlueprint.tokenName}
            className="catalog-page-token-icon"
          />
        ) : null}

        <div>
          <p className="catalog-page-token-name">
            {tokenBlueprint.tokenName}
          </p>
          <p className="catalog-page-token-symbol">
            {tokenBlueprint.symbol}
          </p>
          {tokenBlueprint.description ? (
            <p className="catalog-page-token-description">
              {tokenBlueprint.description}
            </p>
          ) : null}
        </div>
      </div>
    </section>
  );
}