// frontend/console/shell/src/features/list/presentation/components/transportOptionCard.tsx

import { Plus } from "lucide-react";

import type {
  TransportationOption,
} from "../../../../shared/types/inventory";

import { Button } from "../../../../shared/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardSelect,
  CardSelectWrap,
  CardTitle,
} from "../../../../shared/ui/card";
import Stack from "../../../../shared/ui/stack";
import Text from "../../../../shared/ui/text";

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
      <CardHeader>
        <CardTitle>配送方法</CardTitle>

        <Button
          type="button"
          variant="outline"
          onClick={onCreateTransportationFee}
          disabled={loading || disabled}
        >
          <Plus aria-hidden />
          新規登録
        </Button>
      </CardHeader>

      <CardContent>
        <Stack gap="sm">
          {loading ? (
            <Text as="div" size="xs" tone="muted">
              配送方法を読み込み中です…
            </Text>
          ) : selectableOptions.length > 0 ? (
            <CardSelectWrap>
              <CardSelect
                value={selectedValue}
                disabled={disabled}
                onChange={(event) => handleChange(event.target.value)}
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
              </CardSelect>
            </CardSelectWrap>
          ) : (
            <Text as="div" size="xs" tone="muted">
              選択可能な配送方法がありません。
            </Text>
          )}

          {transportationOption === "custom" && transportationId && (
            <Text as="div" size="xs" tone="muted">
              自社配送料金設定を使用します。
            </Text>
          )}
        </Stack>
      </CardContent>
    </Card>
  );
}