// frontend/console/shell/src/pages/locationCreate.tsx

import {
  Card,
  CardContent,
  CardHeader,
  CardInput,
  CardLabel,
  CardTitle,
} from "../shared/ui/card";
import { useLocationCreate } from "../features/company/presentation/hook/useLocationCreate";
import PageStyle from "../layout/PageStyle/PageStyle";

export default function LocationCreate() {
  const {
    vm,
    handlers,
  } = useLocationCreate();

  const disabled = vm.saving;

  return (
    <PageStyle
      layout="single"
      title="在庫保管場所登録"
      onBack={handlers.onBack}
      onSave={handlers.onSave}
      isSaving={vm.saving}
    >
      <div className="mx-auto w-full max-w-3xl">
        <Card>
          <CardContent>
            <div className="space-y-5">
              <div>
                <CardLabel htmlFor="location-name">
                  保管場所名（必須）
                </CardLabel>

                <CardInput
                  id="location-name"
                  name="name"
                  type="text"
                  placeholder="本社倉庫"
                  value={vm.name}
                  onChange={(event) =>
                    handlers.onChangeName(
                      event.target.value,
                    )
                  }
                  disabled={disabled}
                />

                {vm.nameError && (
                  <p className="mt-1 text-xs text-red-500">
                    {vm.nameError}
                  </p>
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
                  value={vm.zipCode}
                  onChange={(event) =>
                    handlers.onChangeZipCode(
                      event.target.value,
                    )
                  }
                  disabled={disabled}
                />

                {vm.zipCodeError && (
                  <p className="mt-1 text-xs text-red-500">
                    {vm.zipCodeError}
                  </p>
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
                  value={vm.state}
                  onChange={(event) =>
                    handlers.onChangeState(
                      event.target.value,
                    )
                  }
                  disabled={disabled}
                />

                {vm.stateError && (
                  <p className="mt-1 text-xs text-red-500">
                    {vm.stateError}
                  </p>
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
                  value={vm.city}
                  onChange={(event) =>
                    handlers.onChangeCity(
                      event.target.value,
                    )
                  }
                  disabled={disabled}
                />

                {vm.cityError && (
                  <p className="mt-1 text-xs text-red-500">
                    {vm.cityError}
                  </p>
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
                  value={vm.street}
                  onChange={(event) =>
                    handlers.onChangeStreet(
                      event.target.value,
                    )
                  }
                  disabled={disabled}
                />

                {vm.streetError && (
                  <p className="mt-1 text-xs text-red-500">
                    {vm.streetError}
                  </p>
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
                  value={vm.street2}
                  onChange={(event) =>
                    handlers.onChangeStreet2(
                      event.target.value,
                    )
                  }
                  disabled={disabled}
                />
              </div>
              {vm.error && (
                <div
                  role="alert"
                  className="rounded-lg border border-red-200 bg-red-50 px-4 py-3"
                >
                  <p className="text-sm text-red-600">
                    {vm.error}
                  </p>
                </div>
              )}

              {vm.saving && (
                <p className="text-sm text-slate-500">
                  在庫保管場所を登録しています...
                </p>
              )}
            </div>
          </CardContent>
        </Card>
      </div>
    </PageStyle>
  );
}