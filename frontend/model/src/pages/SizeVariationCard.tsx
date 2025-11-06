// frontend/model/src/pages/SizeVariationCard.tsx
import * as React from "react";
import { Tags, Trash2 } from "lucide-react";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../shared/ui";
import { Button } from "../../../shared/ui/button";
import { Input } from "../../../shared/ui/input";
import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from "../../../shared/ui/table";
import "./SizeVariationCard.css";
import "./Card.css"

export type SizeRow = {
  id: string;
  sizeLabel: string;
  chest?: number;
  waist?: number;
  length?: number;
  shoulder?: number;
};

type SizeVariationCardProps = {
  sizes: SizeRow[];
  onRemove: (id: string) => void;
};

const SizeVariationCard: React.FC<SizeVariationCardProps> = ({
  sizes,
  onRemove,
}) => {
  return (
    <Card className="svc">
      <CardHeader className="box__header">
        <Tags size={16} />
        <CardTitle className="box__title">サイズバリエーション</CardTitle>
      </CardHeader>

      <CardContent className="box__body">
        <Table className="svc__table">
          <TableHeader>
            <TableRow>
              <TableHead>サイズ</TableHead>
              <TableHead>胸囲(cm)</TableHead>
              <TableHead>ウエスト(cm)</TableHead>
              <TableHead>着丈(cm)</TableHead>
              <TableHead>肩幅(cm)</TableHead>
              <TableHead></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {sizes.map((row) => (
              <TableRow key={row.id}>
                <TableCell>
                  <Input value={row.sizeLabel} variant="readonly" />
                </TableCell>
                <TableCell>
                  <Input value={row.chest ?? ""} variant="readonly" />
                </TableCell>
                <TableCell>
                  <Input value={row.waist ?? ""} variant="readonly" />
                </TableCell>
                <TableCell>
                  <Input value={row.length ?? ""} variant="readonly" />
                </TableCell>
                <TableCell>
                  <Input value={row.shoulder ?? ""} variant="readonly" />
                </TableCell>
                <TableCell>
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => onRemove(row.id)}
                    aria-label={`${row.sizeLabel} を削除`}
                    className="svc__remove"
                  >
                    <Trash2 size={16} />
                  </Button>
                </TableCell>
              </TableRow>
            ))}
            {sizes.length === 0 && (
              <TableRow>
                <TableCell colSpan={6} className="svc__empty">
                  登録されているサイズはありません。
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
};

export default SizeVariationCard;
