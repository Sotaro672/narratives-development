// frontend/console/shell/src/features/transportation/presentation/component/islandFeeEditCard.tsx

import * as React from "react";
import { ChevronDown, MapPin } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "../../../../shared/ui/card";
import { Input } from "../../../../shared/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../../../../shared/ui/table";

import type { TransportationIslandRateVM } from "../../application/transportationService";
import type { IslandCode } from "../../../../shared/types/transporation";

export type IslandFeeEditCardProps = {
  islands: TransportationIslandRateVM[];
  disabled?: boolean;
  className?: string;
  onChangeAmount: (islandCode: IslandCode, amount: string | number | null) => void;
};

const IslandFeeEditCard: React.FC<IslandFeeEditCardProps> = ({
  islands,
  disabled = false,
  className,
  onChangeAmount,
}) => {
  const [isOpen, setIsOpen] = React.useState(false);
  const contentId = "transportation-islands-content";

  const handleToggle = React.useCallback(() => {
    setIsOpen((current) => !current);
  }, []);

  return (
    <Card className={className}>
      <CardHeader>
        <div className="flex items-center gap-2">
          <MapPin size={18} />
          <CardTitle className="text-base">島嶼部</CardTitle>
        </div>

        <button
          type="button"
          onClick={handleToggle}
          aria-expanded={isOpen}
          aria-controls={contentId}
          aria-label={`島嶼部一覧を${isOpen ? "閉じる" : "開く"}`}
          className="flex w-fit items-center gap-2 text-sm font-medium text-slate-600 transition hover:text-slate-900"
        >
          <span className="text-xs font-normal text-slate-400">{islands.length}島</span>
          <ChevronDown
            size={16}
            aria-hidden="true"
            className={`text-slate-500 transition-transform duration-200 ${isOpen ? "rotate-180" : ""}`}
          />
        </button>
      </CardHeader>

      {isOpen && (
        <CardContent id={contentId}>
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>島</TableHead>
                  <TableHead>都道府県</TableHead>
                  <TableHead className="w-48 text-right">送料</TableHead>
                </TableRow>
              </TableHeader>

              <TableBody>
                {islands.map((island) => (
                  <TableRow key={island.islandCode}>
                    <TableCell>
                      <div className="flex flex-col gap-1">
                        <span className="font-medium text-slate-900">{island.islandName}</span>
                        <span className="text-xs text-slate-400">{island.islandCode}</span>
                      </div>
                    </TableCell>

                    <TableCell>
                      <div className="flex items-center gap-2">
                        <span className="text-sm text-slate-700">{island.prefectureName}</span>
                        <span className="text-xs text-slate-400">{island.prefectureCode}</span>
                      </div>
                    </TableCell>

                    <TableCell>
                      <div className="flex items-center justify-end gap-2">
                        <span className="text-sm text-slate-500">¥</span>
                        <Input
                          type="number"
                          inputMode="numeric"
                          min={0}
                          step={1}
                          disabled={disabled}
                          value={island.amount ?? ""}
                          placeholder="未設定"
                          className="h-9 w-32 text-right"
                          aria-label={`${island.islandName}の送料`}
                          onChange={(event) => {
                            onChangeAmount(island.islandCode, event.target.value);
                          }}
                        />
                      </div>
                    </TableCell>
                  </TableRow>
                ))}

                {islands.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={3} className="py-8 text-center text-sm text-slate-500">
                      島嶼部データがありません。
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      )}
    </Card>
  );
};

export default IslandFeeEditCard;