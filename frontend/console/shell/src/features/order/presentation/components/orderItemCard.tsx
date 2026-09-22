// frontend/console/shell/src/features/order/presentation/components/orderItemCard.tsx

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";
import { ColorValue } from "../../../../shared/ui/color";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
} from "../../../../shared/ui/table";
import { coerceRgbInt, rgbIntToHex } from "../../../../shared/util/color";
import { formatJPY } from "../../../../shared/util/currency";
import { safeDateTimeLabelJa } from "../../../../shared/util/dateJa";
import { formatOrderItemValue } from "../formatter/formatOrderItemValue";
import { formatOrderVolume } from "../formatter/formatOrderVolume";
import type { OrderDetailItemDTO } from "../hooks/useOrderDetail";

export type OrderItemCardProps = {
  item: OrderDetailItemDTO;
  index: number;
};

function isAlcoholItem(item: OrderDetailItemDTO): boolean {
  return item.productBlueprintCategoryPath[0] === "alcohol";
}

function getCategoryFieldValue(
  item: OrderDetailItemDTO,
  key: string,
): unknown {
  return item.categoryFields?.[key];
}

export default function OrderItemCard({
  item,
  index,
}: OrderItemCardProps) {
  const transferredAt = safeDateTimeLabelJa(item.transferredAt, "-");
  const alcohol = isAlcoholItem(item);

  const vintage = getCategoryFieldValue(item, "vintage");
  const region = getCategoryFieldValue(item, "region");
  const material = getCategoryFieldValue(item, "material");
  const alcoholContent = getCategoryFieldValue(item, "alcoholContent");

  const colorName = item.color?.trim() ?? "";
  const rgbInt = coerceRgbInt(item.rgb);
  const colorHex = rgbIntToHex(rgbInt);

  return (
    <Card>
      <CardHeader className="order-detail__item-header">
        <CardTitle className="order-detail__item-title">
          アイテム {index + 1}
        </CardTitle>
      </CardHeader>

      <CardContent className="order-detail__item-content">
        <Table className="order-detail__table">
          <TableBody>
            {alcohol ? (
              <>
                <TableRow>
                  <TableHead scope="row" className="order-detail__label-cell">
                    容量
                  </TableHead>
                  <TableCell className="order-detail__value-cell">
                    {formatOrderVolume(item)}
                  </TableCell>
                </TableRow>

                <TableRow>
                  <TableHead scope="row" className="order-detail__label-cell">
                    ヴィンテージ
                  </TableHead>
                  <TableCell className="order-detail__value-cell">
                    {formatOrderItemValue(vintage)}
                  </TableCell>
                </TableRow>

                <TableRow>
                  <TableHead scope="row" className="order-detail__label-cell">
                    地域・産地
                  </TableHead>
                  <TableCell className="order-detail__value-cell">
                    {formatOrderItemValue(region)}
                  </TableCell>
                </TableRow>

                <TableRow>
                  <TableHead scope="row" className="order-detail__label-cell">
                    素材
                  </TableHead>
                  <TableCell className="order-detail__value-cell">
                    {formatOrderItemValue(material)}
                  </TableCell>
                </TableRow>

                <TableRow>
                  <TableHead scope="row" className="order-detail__label-cell">
                    アルコール度数
                  </TableHead>
                  <TableCell className="order-detail__value-cell">
                    {formatOrderItemValue(alcoholContent, "%")}
                  </TableCell>
                </TableRow>
              </>
            ) : (
              <>
                <TableRow>
                  <TableHead scope="row" className="order-detail__label-cell">
                    サイズ
                  </TableHead>
                  <TableCell className="order-detail__value-cell">
                    {item.size ?? "-"}
                  </TableCell>
                </TableRow>

                <TableRow>
                  <TableHead scope="row" className="order-detail__label-cell">
                    カラー
                  </TableHead>
                  <TableCell className="order-detail__value-cell">
                    {!colorName && !colorHex ? (
                      "-"
                    ) : (
                      <ColorValue
                        color={colorHex}
                        size="md"
                        shape="square"
                        swatchTitle={colorHex}
                        swatchAriaLabel={
                          colorHex ? `color ${colorHex}` : undefined
                        }
                      >
                        {colorName || "-"}
                      </ColorValue>
                    )}
                  </TableCell>
                </TableRow>
              </>
            )}

            <TableRow>
              <TableHead scope="row" className="order-detail__label-cell">
                型番
              </TableHead>
              <TableCell className="order-detail__value-cell">
                {item.modelNumber ?? "-"}
              </TableCell>
            </TableRow>

            <TableRow>
              <TableHead scope="row" className="order-detail__label-cell">
                商品名
              </TableHead>
              <TableCell className="order-detail__value-cell">
                {item.productName ?? "-"}
              </TableCell>
            </TableRow>

            <TableRow>
              <TableHead scope="row" className="order-detail__label-cell">
                トークン名
              </TableHead>
              <TableCell className="order-detail__value-cell">
                {item.tokenName ?? "-"}
              </TableCell>
            </TableRow>

            <TableRow>
              <TableHead scope="row" className="order-detail__label-cell">
                数量
              </TableHead>
              <TableCell className="order-detail__value-cell">
                {item.qty}
              </TableCell>
            </TableRow>

            <TableRow>
              <TableHead scope="row" className="order-detail__label-cell">
                金額
              </TableHead>
              <TableCell className="order-detail__value-cell">
                {formatJPY(item.price)}
              </TableCell>
            </TableRow>

            <TableRow>
              <TableHead scope="row" className="order-detail__label-cell">
                移譲日
              </TableHead>
              <TableCell className="order-detail__value-cell">
                {transferredAt}
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}