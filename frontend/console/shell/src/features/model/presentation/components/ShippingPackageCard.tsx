// frontend/console/shell/src/features/model/presentation/components/ShippingPackageCard.tsx

import * as React from "react";
import { Package } from "lucide-react";

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui";
import { Input } from "../../../../shared/ui/input";
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
  ApparelModelNumber,
  ModelVariationMode,
  ShippingPackage,
} from "../../application/modelCreateService";

export type ShippingPackagePatch = Partial<ShippingPackage>;

type CommonShippingPackageCardProps = {
  className?: string;
  mode?: ModelVariationMode;
};

type ApparelShippingPackageCardProps =
  CommonShippingPackageCardProps & {
    kind: "apparel";
    modelNumbers: ApparelModelNumber[];
    onChangeShippingPackage?: (
      size: string,
      color: string,
      patch: ShippingPackagePatch,
    ) => void;
  };

type AlcoholShippingPackageCardProps =
  CommonShippingPackageCardProps & {
    kind: "alcohol";
    modelNumbers: AlcoholModelNumber[];
    onChangeShippingPackage?: (
      volumeLabel: string,
      patch: ShippingPackagePatch,
    ) => void;
  };

export type ShippingPackageCardProps =
  | ApparelShippingPackageCardProps
  | AlcoholShippingPackageCardProps;

type ShippingPackageField = keyof ShippingPackage;

const SHIPPING_PACKAGE_FIELDS: Array<{
  key: ShippingPackageField;
  label: string;
  ariaLabel: string;
}> = [
  {
    key: "weightGrams",
    label: "重量(g)",
    ariaLabel: "梱包後重量",
  },
  {
    key: "widthMm",
    label: "横(mm)",
    ariaLabel: "梱包後横幅",
  },
  {
    key: "lengthMm",
    label: "縦(mm)",
    ariaLabel: "梱包後縦幅",
  },
  {
    key: "heightMm",
    label: "高さ(mm)",
    ariaLabel: "梱包後高さ",
  },
];

function normalizeNumber(value: unknown): number {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return 0;
  }

  return value < 0 ? 0 : Math.floor(value);
}

function normalizeShippingPackage(
  value: ShippingPackage | null | undefined,
): ShippingPackage {
  return {
    weightGrams: normalizeNumber(value?.weightGrams),
    widthMm: normalizeNumber(value?.widthMm),
    lengthMm: normalizeNumber(value?.lengthMm),
    heightMm: normalizeNumber(value?.heightMm),
  };
}

function parseInputNumber(value: string): number {
  if (value === "") {
    return 0;
  }

  const parsed = Number(value);

  if (!Number.isFinite(parsed)) {
    return 0;
  }

  return parsed < 0 ? 0 : Math.floor(parsed);
}

function toAlcoholVolumeLabel(
  modelNumber: AlcoholModelNumber,
): string {
  const value = normalizeNumber(modelNumber.volume.value);
  const unit = String(modelNumber.volume.unit ?? "").trim() || "ml";

  if (value <= 0) {
    return "";
  }

  return `${value}${unit}`;
}

function ShippingPackageInput({
  value,
  label,
  disabled,
  onChange,
}: {
  value: number;
  label: string;
  disabled: boolean;
  onChange: (value: number) => void;
}) {
  if (disabled) {
    return <span>{value > 0 ? value : "-"}</span>;
  }

  return (
    <Input
      type="number"
      min={0}
      step={1}
      inputMode="numeric"
      value={value || ""}
      onChange={(event) =>
        onChange(parseInputNumber(event.target.value))
      }
      aria-label={label}
      placeholder="0"
    />
  );
}

function ApparelShippingPackageRows({
  modelNumbers,
  mode,
  onChangeShippingPackage,
}: {
  modelNumbers: ApparelModelNumber[];
  mode: ModelVariationMode;
  onChangeShippingPackage?: (
    size: string,
    color: string,
    patch: ShippingPackagePatch,
  ) => void;
}) {
  const isEdit = mode === "edit";

  if (modelNumbers.length === 0) {
    return (
      <TableRow>
        <TableCell
          colSpan={7}
          className="mnc__empty"
        >
          登録されているモデルナンバーはありません。
        </TableCell>
      </TableRow>
    );
  }

  return (
    <>
      {modelNumbers.map((modelNumber, index) => {
        const shippingPackage =
          normalizeShippingPackage(modelNumber.shippingPackage);

        const rowKey = [
          modelNumber.size,
          modelNumber.color,
          modelNumber.code,
          index,
        ].join(":");

        return (
          <TableRow key={rowKey}>
            <TableCell className="mnc__size">
              {modelNumber.code || "-"}
            </TableCell>

            <TableCell>
              {modelNumber.size || "-"}
            </TableCell>

            <TableCell>
              {modelNumber.color || "-"}
            </TableCell>

            {SHIPPING_PACKAGE_FIELDS.map((field) => (
              <TableCell key={field.key}>
                <ShippingPackageInput
                  value={shippingPackage[field.key]}
                  label={`${modelNumber.code || "モデル"} ${field.ariaLabel}`}
                  disabled={!isEdit}
                  onChange={(value) =>
                    onChangeShippingPackage?.(
                      modelNumber.size,
                      modelNumber.color,
                      {
                        [field.key]: value,
                      },
                    )
                  }
                />
              </TableCell>
            ))}
          </TableRow>
        );
      })}
    </>
  );
}

function AlcoholShippingPackageRows({
  modelNumbers,
  mode,
  onChangeShippingPackage,
}: {
  modelNumbers: AlcoholModelNumber[];
  mode: ModelVariationMode;
  onChangeShippingPackage?: (
    volumeLabel: string,
    patch: ShippingPackagePatch,
  ) => void;
}) {
  const isEdit = mode === "edit";

  if (modelNumbers.length === 0) {
    return (
      <TableRow>
        <TableCell
          colSpan={6}
          className="mnc__empty"
        >
          登録されているモデルナンバーはありません。
        </TableCell>
      </TableRow>
    );
  }

  return (
    <>
      {modelNumbers.map((modelNumber, index) => {
        const volumeLabel =
          toAlcoholVolumeLabel(modelNumber);

        const shippingPackage =
          normalizeShippingPackage(modelNumber.shippingPackage);

        const rowKey = [
          volumeLabel,
          modelNumber.code,
          index,
        ].join(":");

        return (
          <TableRow key={rowKey}>
            <TableCell className="mnc__size">
              {modelNumber.code || "-"}
            </TableCell>

            <TableCell>
              {volumeLabel || "-"}
            </TableCell>

            {SHIPPING_PACKAGE_FIELDS.map((field) => (
              <TableCell key={field.key}>
                <ShippingPackageInput
                  value={shippingPackage[field.key]}
                  label={`${modelNumber.code || "モデル"} ${field.ariaLabel}`}
                  disabled={!isEdit}
                  onChange={(value) => {
                    if (!volumeLabel) {
                      return;
                    }

                    onChangeShippingPackage?.(
                      volumeLabel,
                      {
                        [field.key]: value,
                      },
                    );
                  }}
                />
              </TableCell>
            ))}
          </TableRow>
        );
      })}
    </>
  );
}

const ShippingPackageCard: React.FC<
  ShippingPackageCardProps
> = (props) => {
  const {
    className,
    mode = "edit",
  } = props;

  const isApparel = props.kind === "apparel";

  return (
    <Card
      className={`spc ${mode === "view" ? "view-mode" : ""} ${
        className ?? ""
      }`}
    >
      <CardHeader className="box__header">
        <Package size={16} />

        <CardTitle className="box__title">
          配送用梱包情報

          {mode === "view" && (
            <span className="ml-2 text-xs text-[var(--pbp-text-soft)]">
              （閲覧）
            </span>
          )}
        </CardTitle>
      </CardHeader>

      <CardContent className="box__body">
        <Table className="mnc__table">
          <TableHeader>
            <TableRow>
              <TableHead>
                モデルナンバー
              </TableHead>

              {isApparel ? (
                <>
                  <TableHead>サイズ</TableHead>
                  <TableHead>カラー</TableHead>
                </>
              ) : (
                <TableHead>容量</TableHead>
              )}

              {SHIPPING_PACKAGE_FIELDS.map((field) => (
                <TableHead key={field.key}>
                  {field.label}
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>

          <TableBody>
            {props.kind === "apparel" ? (
              <ApparelShippingPackageRows
                modelNumbers={props.modelNumbers}
                mode={mode}
                onChangeShippingPackage={
                  props.onChangeShippingPackage
                }
              />
            ) : (
              <AlcoholShippingPackageRows
                modelNumbers={props.modelNumbers}
                mode={mode}
                onChangeShippingPackage={
                  props.onChangeShippingPackage
                }
              />
            )}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
};

export default ShippingPackageCard;