// frontend/console/shell/src/features/order/presentation/components/orderShippingAddress.tsx

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
} from "../../../../shared/ui/table";
import Text from "../../../../shared/ui/text";

export type OrderShippingAddressValue = {
  zipCode?: string | null;
  state?: string | null;
  city?: string | null;
  street?: string | null;
  street2?: string | null;
};

export type OrderShippingAddressProps = {
  shipping?: OrderShippingAddressValue | null;
};

export default function OrderShippingAddress({
  shipping,
}: OrderShippingAddressProps) {
  return (
    <div>
      <Text
        as="div"
        weight="semibold"
        className="order-detail__section-title"
      >
        配送先
      </Text>

      <Table className="order-detail__table">
        <TableBody>
          <TableRow>
            <TableHead
              scope="row"
              className="order-detail__label-cell"
            >
              郵便番号
            </TableHead>
            <TableCell className="order-detail__value-cell order-detail__value-cell--spaced">
              {shipping?.zipCode ?? "-"}
            </TableCell>

            <TableHead
              scope="row"
              className="order-detail__label-cell"
            >
              都道府県
            </TableHead>
            <TableCell className="order-detail__value-cell order-detail__value-cell--spaced">
              {shipping?.state ?? "-"}
            </TableCell>

            <TableHead
              scope="row"
              className="order-detail__label-cell"
            >
              市町村
            </TableHead>
            <TableCell className="order-detail__value-cell">
              {shipping?.city ?? "-"}
            </TableCell>
          </TableRow>

          <TableRow>
            <TableHead
              scope="row"
              className="order-detail__label-cell"
            >
              住所1
            </TableHead>
            <TableCell
              className="order-detail__value-cell"
              colSpan={5}
            >
              {shipping?.street ?? "-"}
            </TableCell>
          </TableRow>

          <TableRow>
            <TableHead
              scope="row"
              className="order-detail__label-cell"
            >
              住所2
            </TableHead>
            <TableCell
              className="order-detail__value-cell"
              colSpan={5}
            >
              {shipping?.street2 ?? "-"}
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </div>
  );
}