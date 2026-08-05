// frontend/amol/src/features/list/presentation/components/ListCard.tsx

import {
  formatPrice,
} from "../../../../components/utils/price";

import type {
  MallListCardItem,
} from "../../types/list";

type ListCardProps = {
  item: MallListCardItem;
  onOpenItem: (
    listId: string,
  ) => void;
};

export default function ListCard({
  item,
  onOpenItem,
}: ListCardProps) {
  const cardTitle =
    item.productName?.trim() ||
    item.title.trim();

  const cardBrandName =
    item.brandName?.trim() ?? "";

  const firstPrice =
    Array.isArray(item.prices)
      ? item.prices[0]
      : undefined;

  const priceAmount =
    firstPrice?.amount ??
    firstPrice?.price;

  const formattedPrice =
    formatPrice(
      priceAmount,
      {
        currency:
          firstPrice?.currency,
      },
    );

  function handleClick() {
    const listId =
      item.id.trim();

    if (!listId) {
      return;
    }

    onOpenItem(listId);
  }

  return (
    <button
      type="button"
      className="lists-page-card"
      onClick={handleClick}
    >
      <div className="lists-page-card-image-wrap">
        {item.image ? (
          <img
            src={item.image}
            alt={cardTitle}
            className="lists-page-card-image"
            loading="lazy"
          />
        ) : (
          <div className="lists-page-card-image-placeholder">
            No Image
          </div>
        )}
      </div>

      <div className="lists-page-card-body">
        <h2 className="lists-page-card-title">
          {cardTitle}
        </h2>

        {cardBrandName ? (
          <p className="lists-page-card-description">
            {cardBrandName}
          </p>
        ) : null}

        <div className="lists-page-card-footer">
          <span className="lists-page-card-price">
            {formattedPrice}
          </span>
        </div>
      </div>
    </button>
  );
}