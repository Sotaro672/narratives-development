// frontend/console/shell/src/features/model/presentation/components/ShippingPackageCard.tsx

import * as React from "react";
import { Package } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "../../../../shared/ui";
import { Input } from "../../../../shared/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../../../../shared/ui/table";

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

type ApparelShippingPackageCardProps = CommonShippingPackageCardProps & {
  kind: "apparel";
  modelNumbers: ApparelModelNumber[];
  onChangeShippingPackage?: (size: string, patch: ShippingPackagePatch) => void;
};

type AlcoholShippingPackageCardProps = CommonShippingPackageCardProps & {
  kind: "alcohol";
  modelNumbers: AlcoholModelNumber[];
  onChangeShippingPackage?: (volumeLabel: string, patch: ShippingPackagePatch) => void;
};

export type ShippingPackageCardProps =
  | ApparelShippingPackageCardProps
  | AlcoholShippingPackageCardProps;

type ShippingPackageField = keyof ShippingPackage;

type ShippingPackageFieldDefinition = {
  key: ShippingPackageField;
  label: string;
  ariaLabel: string;
  unit: "g" | "cm";
};

const SHIPPING_PACKAGE_FIELDS: ShippingPackageFieldDefinition[] = [
  {
    key: "weightGrams",
    label: "重量(g)",
    ariaLabel: "梱包後重量",
    unit: "g",
  },
  {
    key: "widthMm",
    label: "横(cm)",
    ariaLabel: "梱包後横幅",
    unit: "cm",
  },
  {
    key: "lengthMm",
    label: "縦(cm)",
    ariaLabel: "梱包後縦幅",
    unit: "cm",
  },
  {
    key: "heightMm",
    label: "高さ(cm)",
    ariaLabel: "梱包後高さ",
    unit: "cm",
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

/**
 * 内部値をConsole表示用の値へ変換する。
 *
 * 重量:
 *   g → g
 *
 * 寸法:
 *   mm → cm
 *
 * 例:
 *   245mm → 24.5cm
 *   300mm → 30cm
 */
function toDisplayValue(
  value: number,
  unit: ShippingPackageFieldDefinition["unit"],
): number {
  const normalized = normalizeNumber(value);

  if (unit === "cm") {
    return normalized / 10;
  }

  return normalized;
}

/**
 * Consoleの入力値を内部保存用の値へ変換する。
 *
 * 重量:
 *   g → g整数
 *
 * 寸法:
 *   cm → mm整数
 *
 * 例:
 *   24.5cm → 245mm
 *   30cm   → 300mm
 */
function parseInputValue(
  value: string,
  unit: ShippingPackageFieldDefinition["unit"],
): number {
  if (value === "") {
    return 0;
  }

  const parsed = Number(value);

  if (!Number.isFinite(parsed) || parsed < 0) {
    return 0;
  }

  if (unit === "cm") {
    return Math.round(parsed * 10);
  }

  return Math.floor(parsed);
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
  field,
  label,
  disabled,
  onChange,
}: {
  value: number;
  field: ShippingPackageFieldDefinition;
  label: string;
  disabled: boolean;
  onChange: (value: number) => void;
}) {
  const displayValue = toDisplayValue(value, field.unit);

  if (disabled) {
    return <span>{displayValue > 0 ? displayValue : "-"}</span>;
  }

  const isDimension = field.unit === "cm";

  return (
    <Input
      type="number"
      min={0}
      step={isDimension ? 0.1 : 1}
      inputMode={isDimension ? "decimal" : "numeric"}
      value={displayValue || ""}
      onChange={(event) =>
        onChange(
          parseInputValue(
            event.target.value,
            field.unit,
          ),
        )
      }
      aria-label={`${label} (${field.unit})`}
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
    patch: ShippingPackagePatch,
  ) => void;
}) {
  const isEdit = mode === "edit";

  const sizeRows = React.useMemo(() => {
    const rows = new Map<string, ApparelModelNumber>();

    for (const modelNumber of modelNumbers) {
      const size = String(modelNumber.size ?? "").trim();

      if (!size || rows.has(size)) {
        continue;
      }

      rows.set(size, modelNumber);
    }

    return Array.from(rows.entries());
  }, [modelNumbers]);

  if (sizeRows.length === 0) {
    return (
      <TableRow>
        <TableCell colSpan={5} className="mnc__empty">
          登録されているサイズはありません。
        </TableCell>
      </TableRow>
    );
  }

  return (
    <>
      {sizeRows.map(([size, modelNumber]) => {
        const shippingPackage =
          normalizeShippingPackage(modelNumber.shippingPackage);

        return (
          <TableRow key={size}>
            <TableCell className="mnc__size">{size}</TableCell>

            {SHIPPING_PACKAGE_FIELDS.map((field) => (
              <TableCell key={field.key}>
                <ShippingPackageInput
                  value={shippingPackage[field.key]}
                  field={field}
                  label={`${size} ${field.ariaLabel}`}
                  disabled={!isEdit}
                  onChange={(value) =>
                    onChangeShippingPackage?.(size, {
                      [field.key]: value,
                    })
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
        <TableCell colSpan={5} className="mnc__empty">
          登録されている容量はありません。
        </TableCell>
      </TableRow>
    );
  }

  return (
    <>
      {modelNumbers.map((modelNumber, index) => {
        const volumeLabel = toAlcoholVolumeLabel(modelNumber);
        const shippingPackage =
          normalizeShippingPackage(modelNumber.shippingPackage);
        const rowKey = [volumeLabel, index].join(":");

        return (
          <TableRow key={rowKey}>
            <TableCell className="mnc__size">
              {volumeLabel || "-"}
            </TableCell>

            {SHIPPING_PACKAGE_FIELDS.map((field) => (
              <TableCell key={field.key}>
                <ShippingPackageInput
                  value={shippingPackage[field.key]}
                  field={field}
                  label={`${volumeLabel || "容量"} ${field.ariaLabel}`}
                  disabled={!isEdit}
                  onChange={(value) => {
                    if (!volumeLabel) {
                      return;
                    }

                    onChangeShippingPackage?.(volumeLabel, {
                      [field.key]: value,
                    });
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

const ShippingPackageCard: React.FC<ShippingPackageCardProps> = (props) => {
  const { className, mode = "edit" } = props;
  const isApparel = props.kind === "apparel";

  return (
    <Card
      className={`spc ${mode === "view" ? "view-mode" : ""} ${className ?? ""}`}
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
              <TableHead>{isApparel ? "サイズ" : "容量"}</TableHead>

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
                onChangeShippingPackage={props.onChangeShippingPackage}
              />
            ) : (
              <AlcoholShippingPackageRows
                modelNumbers={props.modelNumbers}
                mode={mode}
                onChangeShippingPackage={props.onChangeShippingPackage}
              />
            )}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
};

export default ShippingPackageCard;