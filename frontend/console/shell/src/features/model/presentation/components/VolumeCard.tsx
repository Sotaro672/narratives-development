// frontend/console/model/src/presentation/components/VolumeCard.tsx

import * as React from "react";
import { Beaker, Plus, Trash2 } from "lucide-react";

import {
  Card,
  CardContent,
  CardHeader,
  CardHeaderIcon,
  CardHeaderLeft,
  CardInput,
  CardReadonly,
  CardTitle,
} from "../../../../shared/ui";
import { Button } from "../../../../shared/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../../../../shared/ui/table";

import type { VolumeRow } from "../../application/modelCreateService";

type VolumePatch = Partial<Omit<VolumeRow, "id">>;

type VolumeCardProps = {
  volumes: VolumeRow[];
  className?: string;
  mode?: "edit" | "view";
  onAddVolume?: () => void;
  onRemoveVolume?: (id: string) => void;
  onChangeVolume?: (id: string, patch: VolumePatch) => void;
};

function toInputNumberValue(value: number | undefined): string {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return "";
  }

  return String(value);
}

const VolumeCard: React.FC<VolumeCardProps> = ({
  volumes,
  className,
  mode = "edit",
  onAddVolume,
  onRemoveVolume,
  onChangeVolume,
}) => {
  const isEdit = mode === "edit";

  const handleChangeVolumeValue =
    (id: string) => (event: React.ChangeEvent<HTMLInputElement>) => {
      if (!isEdit) {
        return;
      }

      const rawValue = event.target.value;

      if (rawValue.trim() === "") {
        onChangeVolume?.(id, {
          volumeValue: 0,
        });
        return;
      }

      const nextValue = Number(rawValue);

      onChangeVolume?.(id, {
        volumeValue: Number.isFinite(nextValue) ? nextValue : 0,
      });
    };

  const handleChangeVolumeUnit =
    (id: string) => (event: React.ChangeEvent<HTMLInputElement>) => {
      if (!isEdit) {
        return;
      }

      onChangeVolume?.(id, {
        volumeUnit: event.target.value,
      });
    };

  return (
    <Card
      className={`vc-volume ${className ?? ""} ${
        mode === "view" ? "view-mode" : ""
      }`}
    >
      <CardHeader>
        <CardHeaderLeft>
          <CardHeaderIcon>
            <Beaker className="card__header-icon-svg" />
          </CardHeaderIcon>

          <CardTitle strong>
            容量

            {mode === "view" && (
              <span className="ml-2 text-xs text-[var(--pbp-text-soft)]">
                （閲覧）
              </span>
            )}
          </CardTitle>
        </CardHeaderLeft>

        {isEdit && (
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={onAddVolume}
            aria-label="容量を追加"
          >
            <Plus size={14} />
            追加
          </Button>
        )}
      </CardHeader>

      <CardContent>
        <Table className="svc__table">
          <TableHeader>
            <TableRow>
              <TableHead>容量</TableHead>
              <TableHead>単位</TableHead>
              {isEdit && <TableHead>操作</TableHead>}
            </TableRow>
          </TableHeader>

          <TableBody>
            {volumes.map((volume) => (
              <TableRow key={volume.id}>
                <TableCell>
                  {isEdit ? (
                    <CardInput
                      type="number"
                      min={0}
                      value={toInputNumberValue(volume.volumeValue)}
                      onChange={handleChangeVolumeValue(volume.id)}
                      placeholder="例: 720"
                      aria-label="容量"
                    />
                  ) : (
                    <CardReadonly inputLike aria-label="容量">
                      {toInputNumberValue(volume.volumeValue) || "-"}
                    </CardReadonly>
                  )}
                </TableCell>

                <TableCell>
                  {isEdit ? (
                    <CardInput
                      value={volume.volumeUnit}
                      onChange={handleChangeVolumeUnit(volume.id)}
                      placeholder="ml"
                      aria-label="容量の単位"
                    />
                  ) : (
                    <CardReadonly inputLike aria-label="容量の単位">
                      {volume.volumeUnit || "-"}
                    </CardReadonly>
                  )}
                </TableCell>

                {isEdit && (
                  <TableCell>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="svc__remove"
                      onClick={() => onRemoveVolume?.(volume.id)}
                      aria-label="容量を削除"
                    >
                      <Trash2 size={14} />
                    </Button>
                  </TableCell>
                )}
              </TableRow>
            ))}

            {volumes.length === 0 && (
              <TableRow>
                <TableCell
                  colSpan={isEdit ? 3 : 2}
                  className="svc__empty"
                >
                  登録されている容量はありません。
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
};

export default VolumeCard;