// frontend/console/shell/src/features/company/presentation/components/LocationFormFields.tsx

import {
  CardInput,
  CardLabel,
} from "../../../../shared/ui/card";
import { ErrorMessage } from "../../../../shared/ui/error";
import Stack from "../../../../shared/ui/stack";
import Text from "../../../../shared/ui/text";

import "../../../../styles/location.css";

export type LocationFormValue = {
  name: string;
  zipCode: string;
  state: string;
  city: string;
  street: string;
  street2: string;
};

export type LocationFormErrors = {
  name: string | null;
  zipCode: string | null;
  state: string | null;
  city: string | null;
  street: string | null;
};

export type LocationFormFieldsProps = {
  value: LocationFormValue;
  errors: LocationFormErrors;
  disabled?: boolean;
  addressSearching?: boolean;
  onChangeName: (value: string) => void;
  onChangeZipCode: (value: string) => void;
  onChangeState: (value: string) => void;
  onChangeCity: (value: string) => void;
  onChangeStreet: (value: string) => void;
  onChangeStreet2: (value: string) => void;
};

export default function LocationFormFields({
  value,
  errors,
  disabled = false,
  addressSearching = false,
  onChangeName,
  onChangeZipCode,
  onChangeState,
  onChangeCity,
  onChangeStreet,
  onChangeStreet2,
}: LocationFormFieldsProps) {
  return (
    <Stack gap="lg">
      <div>
        <CardLabel htmlFor="location-name">
          保管場所名（必須）
        </CardLabel>

        <CardInput
          id="location-name"
          name="name"
          type="text"
          placeholder="本社倉庫"
          value={value.name}
          onChange={(event) => onChangeName(event.target.value)}
          disabled={disabled}
        />

        {errors.name && (
          <ErrorMessage
            as="p"
            size="xs"
            className="location-form-fields__feedback"
          >
            {errors.name}
          </ErrorMessage>
        )}
      </div>

      <div>
        <CardLabel htmlFor="location-zip-code">
          郵便番号（必須）
        </CardLabel>

        <CardInput
          id="location-zip-code"
          name="zipCode"
          type="text"
          inputMode="numeric"
          autoComplete="postal-code"
          placeholder="100-0001"
          value={value.zipCode}
          onChange={(event) => onChangeZipCode(event.target.value)}
          disabled={disabled}
        />

        {addressSearching && (
          <Text
            as="p"
            size="xs"
            tone="muted"
            className="location-form-fields__feedback"
            role="status"
            aria-live="polite"
          >
            住所を検索しています...
          </Text>
        )}

        {!addressSearching && errors.zipCode && (
          <ErrorMessage
            as="p"
            size="xs"
            className="location-form-fields__feedback"
          >
            {errors.zipCode}
          </ErrorMessage>
        )}
      </div>

      <div>
        <CardLabel htmlFor="location-state">
          都道府県（必須）
        </CardLabel>

        <CardInput
          id="location-state"
          name="state"
          type="text"
          autoComplete="address-level1"
          placeholder="東京都"
          value={value.state}
          onChange={(event) => onChangeState(event.target.value)}
          disabled={disabled}
        />

        {errors.state && (
          <ErrorMessage
            as="p"
            size="xs"
            className="location-form-fields__feedback"
          >
            {errors.state}
          </ErrorMessage>
        )}
      </div>

      <div>
        <CardLabel htmlFor="location-city">
          市区町村（必須）
        </CardLabel>

        <CardInput
          id="location-city"
          name="city"
          type="text"
          autoComplete="address-level2"
          placeholder="千代田区"
          value={value.city}
          onChange={(event) => onChangeCity(event.target.value)}
          disabled={disabled}
        />

        {errors.city && (
          <ErrorMessage
            as="p"
            size="xs"
            className="location-form-fields__feedback"
          >
            {errors.city}
          </ErrorMessage>
        )}
      </div>

      <div>
        <CardLabel htmlFor="location-street">
          住所（必須）
        </CardLabel>

        <CardInput
          id="location-street"
          name="street"
          type="text"
          autoComplete="address-line1"
          placeholder="千代田1-1"
          value={value.street}
          onChange={(event) => onChangeStreet(event.target.value)}
          disabled={disabled}
        />

        {errors.street && (
          <ErrorMessage
            as="p"
            size="xs"
            className="location-form-fields__feedback"
          >
            {errors.street}
          </ErrorMessage>
        )}
      </div>

      <div>
        <CardLabel htmlFor="location-street2">
          建物名・部屋番号
        </CardLabel>

        <CardInput
          id="location-street2"
          name="street2"
          type="text"
          autoComplete="address-line2"
          placeholder="AMOLビル 3F"
          value={value.street2}
          onChange={(event) => onChangeStreet2(event.target.value)}
          disabled={disabled}
        />
      </div>
    </Stack>
  );
}