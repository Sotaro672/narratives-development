// frontend/console/shell/src/features/order/presentation/components/orderBuyerCard.tsx

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
} from "../../../../shared/ui/table";
import Text from "../../../../shared/ui/text";

export type OrderBuyerCardProps = {
  loading: boolean;
  error: string | null;
  orderExists: boolean;
  userName: string;
  email: string;
};

export default function OrderBuyerCard({
  loading,
  error,
  orderExists,
  userName,
  email,
}: OrderBuyerCardProps) {
  return (
    <div className="page-column page-column--offset-top sticky-aside">
      <Card>
        <CardHeader>
          <CardTitle className="order-detail__card-title">
            購入者情報
          </CardTitle>
        </CardHeader>

        <CardContent>
          {loading ? (
            <Text
              as="div"
              tone="muted"
              className="order-detail__message"
            >
              読み込み中...
            </Text>
          ) : error ? (
            <Text
              as="div"
              tone="destructive"
              wrap="pre-wrap"
              className="order-detail__message"
              role="alert"
            >
              {error}
            </Text>
          ) : !orderExists ? (
            <Text
              as="div"
              tone="muted"
              className="order-detail__message"
            >
              -
            </Text>
          ) : (
            <Table className="order-detail__table">
              <TableBody>
                <TableRow>
                  <TableHead
                    scope="row"
                    className="order-detail__label-cell"
                  >
                    ユーザー名
                  </TableHead>
                  <TableCell className="order-detail__value-cell">
                    {userName}
                  </TableCell>
                </TableRow>

                <TableRow>
                  <TableHead
                    scope="row"
                    className="order-detail__label-cell"
                  >
                    メールアドレス
                  </TableHead>
                  <TableCell className="order-detail__value-cell">
                    {email}
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}