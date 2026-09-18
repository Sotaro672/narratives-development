// frontend/console/model/presentation/components/AlcoholModelNumberCard.tsx

import * as React from "react";
import { Tags } from "lucide-react";

import {
  Card,
  CardContent,
  CardHeader,
  CardHeaderIcon,
  CardHeaderLeft,
  CardInput,
  CardTitle,
} from "../../../../shared/ui";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../../../../shared/ui/table";

import type {
  AlcoholModelNumber,
  VolumeRow,
} from "../../application/modelCreateService";

type AlcoholModelNumberCardProps = {
  volumes: VolumeRow[];
  modelNumbers: AlcoholModelNumber[];
  className?: string;
  mode?: "edit" | "view";
  onChangeModelNumber?: (volumeLabel: string, nextCode: string) => void;
};

function toVolumeLabel(
  volume: Pick<VolumeRow, "volumeValue" | "volumeUnit">,
): string {
  const value =
    typeof volume.volumeValue === "number" &&
    Number.isFinite(volume.volumeValue)
      ? volume.volumeValue
      : 0;

  const unit = String(volume.volumeUnit ?? "").trim() || "ml";

  if (value <= 0) {
    return "";
  }

  return `${value}${unit}`;
}

function toAlcoholModelNumberVolumeLabel(
  modelNumber: AlcoholModelNumber,
): string {
  const value =
    typeof modelNumber.volume.value === "number" &&
    Number.isFinite(modelNumber.volume.value)
      ? modelNumber.volume.value
      : 0;

  const unit = String(modelNumber.volume.unit ?? "").trim() || "ml";

  if (value <= 0) {
    return "";
  }

  return `${value}${unit}`;
}

function getCode(
  modelNumbers: AlcoholModelNumber[],
  volumeLabel: string,
): string {
  const found = modelNumbers.find(
    (modelNumber) =>
      toAlcoholModelNumberVolumeLabel(modelNumber) === volumeLabel,
  );

  return found?.code ?? "";
}

const AlcoholModelNumberCard: React.FC<AlcoholModelNumberCardProps> = ({
  volumes,
  modelNumbers,
  className,
  mode = "edit",
  onChangeModelNumber,
}) => {
  const isEdit = mode === "edit";

  const visibleVolumes = React.useMemo(
    () =>
      volumes
        .map((volume) => ({
          ...volume,
          volumeLabel: toVolumeLabel(volume),
        }))
        .filter((volume) => volume.volumeLabel),
    [volumes],
  );

  const handleChange =
    (volumeLabel: string) =>
    (event: React.ChangeEvent<HTMLInputElement>) => {
      if (!isEdit) {
        return;
      }

      onChangeModelNumber?.(volumeLabel, event.target.value);
    };

  return (
    <Card
      className={`amnc ${className ?? ""} ${
        mode === "view" ? "view-mode" : ""
      }`}
    >
      <CardHeader>
        <CardHeaderLeft>
          <CardHeaderIcon>
            <Tags className="card__header-icon-svg" />
          </CardHeaderIcon>

          <CardTitle strong>
            容量別モデルナンバー

            {mode === "view" && (
              <span className="ml-2 text-xs text-[var(--pbp-text-soft)]">
                （閲覧）
              </span>
            )}
          </CardTitle>
        </CardHeaderLeft>
      </CardHeader>

      <CardContent>
        <Table className="mnc__table">
          <TableHeader>
            <TableRow>
              <TableHead>容量</TableHead>
              <TableHead>モデルナンバー</TableHead>
            </TableRow>
          </TableHeader>

          <TableBody>
            {visibleVolumes.map((volume) => {
              const code = getCode(
                modelNumbers,
                volume.volumeLabel,
              );

              return (
                <TableRow key={volume.id}>
                  <TableCell className="mnc__size">
                    {volume.volumeLabel}
                  </TableCell>

                  <TableCell>
                    {isEdit ? (
                      <CardInput
                        value={code}
                        onChange={handleChange(volume.volumeLabel)}
                        placeholder="例: SAKE-720"
                        aria-label={`${volume.volumeLabel} のモデルナンバー`}
                      />
                    ) : (
                      <span>{code}</span>
                    )}
                  </TableCell>
                </TableRow>
              );
            })}

            {visibleVolumes.length === 0 && (
              <TableRow>
                <TableCell
                  colSpan={2}
                  className="mnc__empty"
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

export default AlcoholModelNumberCard;