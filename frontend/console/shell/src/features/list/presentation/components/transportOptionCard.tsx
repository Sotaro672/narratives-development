// frontend/console/shell/src/features/list/presentation/components/transportOptionCard.tsx

import { Plus } from "lucide-react";

import type {
  TransportationOption,
} from "../../../../shared/types/inventory";

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";
import Stack from "../../../../shared/ui/stack";

import "../../../../styles/transportation.css";

export type TransportOptionCardOption = {
  transportationOption: TransportationOption;
  transportationId?: string;
  name: string;
};

type TransportOptionCardProps = {
  options: TransportOptionCardOption[];
  transportationOption: TransportationOption | "";
  transportationId: string;
  onSelectTransportationOption: (value: string) => void;
  setTransportationId: (value: string) => void;
  onCreateTransportationFee: () => void;
  loading?: boolean;
  disabled?: boolean;
};

function buildOptionValue(
  option: TransportOptionCardOption,
): string {
  if (option.transportationOption === "custom") {
    if (!option.transportationId) {
      return "";
    }

    return `custom:${option.transportationId}`;
  }

  return option.transportationOption;
}

function buildSelectedValue(
  transportationOption: TransportationOption | "",
  transportationId: string,
): string {
  if (!transportationOption) {
    return "";
  }

  if (transportationOption === "custom") {
    if (!transportationId) {
      return "";
    }

    return `custom:${transportationId}`;
  }

  return transportationOption;
}

export default function TransportOptionCard({
  options,
  transportationOption,
  transportationId,
  onSelectTransportationOption,
  setTransportationId,
  onCreateTransportationFee,
  loading = false,
  disabled = false,
}: TransportOptionCardProps) {
  const selectedValue = buildSelectedValue(
    transportationOption,
    transportationId,
  );

  const selectableOptions = options.filter((option) => {
    if (option.transportationOption !== "custom") {
      return true;
    }

    return Boolean(option.transportationId);
  });

  const handleChange = (value: string) => {
    if (disabled) {
      return;
    }

    if (!value) {
      onSelectTransportationOption("");
      setTransportationId("");
      return;
    }

    const selectedOption = selectableOptions.find(
      (option) => buildOptionValue(option) === value,
    );

    if (!selectedOption) {
      onSelectTransportationOption("");
      setTransportationId("");
      return;
    }

    onSelectTransportationOption(
      selectedOption.transportationOption,
    );

    if (selectedOption.transportationOption === "custom") {
      setTransportationId(
        selectedOption.transportationId ?? "",
      );
      return;
    }

    setTransportationId("");
  };

  return (
    <Card>
      <CardHeader className="transport-option-card__header">
        <CardTitle>配送方法</CardTitle>

        <button
          type="button"
          className="transport-option-card__create-button"
          onClick={onCreateTransportationFee}
          disabled={loading || disabled}
        >
          <Plus size={16} aria-hidden />
          新規登録
        </button>
      </CardHeader>

      <CardContent>
        <Stack gap="sm">
          {loading ? (
            <div className="transport-option-card__state">
              配送方法を読み込み中です…
            </div>
          ) : selectableOptions.length > 0 ? (
            <select
              value={selectedValue}
              disabled={disabled}
              onChange={(event) => handleChange(event.target.value)}
              className="transport-option-card__select"
            >
              <option value="">
                配送方法を選択してください
              </option>

              {selectableOptions.map((option) => {
                const value = buildOptionValue(option);

                return (
                  <option key={value} value={value}>
                    {option.name}
                  </option>
                );
              })}
            </select>
          ) : (
            <div className="transport-option-card__state">
              選択可能な配送方法がありません。
            </div>
          )}

          {transportationOption === "custom" && transportationId && (
            <div className="transport-option-card__note">
              自社配送料金設定を使用します。
            </div>
          )}
        </Stack>
      </CardContent>
    </Card>
  );
}