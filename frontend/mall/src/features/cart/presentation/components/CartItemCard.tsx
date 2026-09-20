// frontend/mall/src/features/cart/presentation/components/CartItemCard.tsx

import Card from "../../../../components/ui/Card";
import IconButton from "../../../../components/ui/IconButton";
import Media from "../../../../components/ui/Media";
import { formatPrice } from "../../../../components/utils/price";
import type { CartDisplayItem } from "../../../shared/types/cart";
import {
  getCartItemBrandName,
  getCartItemImageUrl,
  getCartItemListTitle,
  getCartItemNavigationPath,
  getCartItemPrice,
  getCartItemProductName,
} from "../../utils/cartUtils";
import CartItemMeta from "./CartItemMeta";

type CartItemCardProps = {
  item: CartDisplayItem;
  removing: boolean;
  removalDisabled: boolean;
  onRemove: (item: CartDisplayItem) => void | Promise<void>;
  onOpen: (path: string) => void;
};

export default function CartItemCard({
  item,
  removing,
  removalDisabled,
  onRemove,
  onOpen,
}: CartItemCardProps) {
  const brandName = getCartItemBrandName(item);
  const productName = getCartItemProductName(item);
  const listTitle = getCartItemListTitle(item);
  const imageUrl = getCartItemImageUrl(item);
  const navigationPath = getCartItemNavigationPath(item);
  const canNavigate = navigationPath.length > 0;
  const price = getCartItemPrice(item);
  const lineAmount = price === null ? null : price * item.qty;

  function handleOpen() {
    if (!canNavigate) {
      return;
    }

    onOpen(navigationPath);
  }

  function handleRemove() {
    if (removalDisabled || removing) {
      return;
    }

    void onRemove(item);
  }

  return (
    <Card as="article" variant="panel" className="cart-page-item">
      <IconButton
        type="button"
        variant="danger"
        size="sm"
        className="cart-page-item__remove-button"
        aria-label={`${productName}をカートから削除`}
        aria-busy={removing}
        disabled={removalDisabled || removing}
        onClick={(event) => {
          event.stopPropagation();
          handleRemove();
        }}
      >
        {removing ? "…" : "×"}
      </IconButton>

      <button
        type="button"
        className="cart-page-item__image-button"
        aria-label={canNavigate ? `${productName}の詳細を見る` : undefined}
        disabled={!canNavigate}
        onClick={handleOpen}
      >
        {imageUrl ? (
          <Media
            src={imageUrl}
            alt={productName}
            loading="lazy"
            fit="cover"
          />
        ) : (
          <div className="cart-page-item__image-placeholder">No Image</div>
        )}
      </button>

      <div className="cart-page-item__body">
        <p className="cart-page-item__brand">{brandName}</p>
        <h2 className="cart-page-item__title">{productName}</h2>

        {listTitle ? (
          <p className="cart-page-item__list-title">{listTitle}</p>
        ) : null}

        <CartItemMeta item={item} />
        <p className="cart-page-item__price">{formatPrice(lineAmount)}</p>
      </div>
    </Card>
  );
}