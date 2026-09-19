// frontend/console/shell/src/features/order/presentation/components/orderBasicInfo.tsx

import Link from "../../../../shared/ui/link";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
} from "../../../../shared/ui/table";
import Text from "../../../../shared/ui/text";
import { formatJPY } from "../../../../shared/util/currency";

export type OrderBasicInfoList = {
  id: string;
  readableId: string;
};

export type OrderBasicInfoProps = {
  createdAt: string;
  lists: OrderBasicInfoList[];
  itemCount: number;
  quantity: number;
  subtotal: number;
  shippingAmount: number;
  consumptionTax: number;
  totalPrice: number;
  onListClick: (listId: string) => void;
};

export default function OrderBasicInfo({
  createdAt,
  lists,
  itemCount,
  quantity,
  subtotal,
  shippingAmount,
  consumptionTax,
  totalPrice,
  onListClick,
}: OrderBasicInfoProps) {
  return (
    <div>
      <Text
        as="div"
        weight="semibold"
        className="order-detail__section-title"
      >
        基本情報
      </Text>

      <Table className="order-detail__table">
        <TableBody>
          <TableRow>
            <TableHead scope="row" className="order-detail__label-cell">
              注文日
            </TableHead>
            <TableCell className="order-detail__value-cell">
              {createdAt}
            </TableCell>
          </TableRow>

          <TableRow>
            <TableHead scope="row" className="order-detail__label-cell">
              リストID
            </TableHead>
            <TableCell className="order-detail__value-cell">
              {lists.length > 0 ? (
                <div className="order-detail__list-links">
                  {lists.map((list) => (
                    <Link
                      key={list.id}
                      onClick={() => onListClick(list.id)}
                    >
                      {list.readableId}
                    </Link>
                  ))}
                </div>
              ) : (
                "-"
              )}
            </TableCell>
          </TableRow>

          <TableRow>
            <TableHead scope="row" className="order-detail__label-cell">
              アイテム数
            </TableHead>
            <TableCell className="order-detail__value-cell">
              {itemCount} 点
            </TableCell>
          </TableRow>

          <TableRow>
            <TableHead scope="row" className="order-detail__label-cell">
              数量合計
            </TableHead>
            <TableCell className="order-detail__value-cell">
              {quantity} 点
            </TableCell>
          </TableRow>

          <TableRow>
            <TableHead scope="row" className="order-detail__label-cell">
              商品小計
            </TableHead>
            <TableCell className="order-detail__value-cell">
              {formatJPY(subtotal)}
            </TableCell>
          </TableRow>

          <TableRow>
            <TableHead scope="row" className="order-detail__label-cell">
              配送料
            </TableHead>
            <TableCell className="order-detail__value-cell">
              {formatJPY(shippingAmount)}
            </TableCell>
          </TableRow>

          <TableRow>
            <TableHead scope="row" className="order-detail__label-cell">
              消費税
            </TableHead>
            <TableCell className="order-detail__value-cell">
              {formatJPY(consumptionTax)}
            </TableCell>
          </TableRow>

          <TableRow>
            <TableHead scope="row" className="order-detail__label-cell">
              合計金額
            </TableHead>
            <TableCell className="order-detail__value-cell">
              {formatJPY(totalPrice)}
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </div>
  );
}