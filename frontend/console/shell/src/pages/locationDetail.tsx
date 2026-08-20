// frontend/console/shell/src/pages/locationDetail.tsx

import { useCallback, useState } from "react";
import { useParams } from "react-router-dom";

import { useLocationDetail } from "../features/company/presentation/hook/useLocationDetail";
import PageStyle from "../layout/PageStyle/PageStyle";
import {
  Card,
  CardContent,
  CardHeader,
  CardInput,
  CardLabel,
  CardTitle,
} from "../shared/ui/card";
import { safeDateTimeLabelJa } from "../shared/util/dateJa";

export default function LocationDetail() {
  const { locationId } = useParams<{
    locationId: string;
  }>();

  const {
    vm,
    handlers,
  } = useLocationDetail(locationId);

  const [isEditing, setIsEditing] = useState(false);

  const location = vm.location;

  const handleEdit = useCallback(() => {
    setIsEditing(true);
  }, []);

  const handleCancel = useCallback(() => {
    handlers.onReset();
    setIsEditing(false);
  }, [handlers]);

  const handleSave = useCallback(async () => {
    const saved = await handlers.onSave();

    if (saved) {
      setIsEditing(false);
    }
  }, [handlers]);

  const handleDelete = useCallback(async () => {
    await handlers.onDelete();
  }, [handlers]);

  const disabled =
    !isEditing ||
    vm.saving ||
    vm.deleting;

  const createdAt = location
    ? safeDateTimeLabelJa(
        location.createdAt,
        "",
      )
    : "";

  const updatedAt = location
    ? safeDateTimeLabelJa(
        location.updatedAt,
        "",
      )
    : "";

  return (
    <PageStyle
      layout="single"
      title="在庫保管場所詳細"
      onBack={handlers.onBack}
      onEdit={
        !isEditing && location
          ? handleEdit
          : undefined
      }
      onDelete={
        isEditing
          ? handleDelete
          : undefined
      }
      onCancel={
        isEditing
          ? handleCancel
          : undefined
      }
      onSave={
        isEditing
          ? handleSave
          : undefined
      }
      isSaving={vm.saving}
    >
      <div className="mx-auto w-full max-w-3xl">
        {vm.loading ? (
          <div className="rounded-lg border border-slate-200 bg-white px-5 py-12 text-center text-sm text-slate-500">
            在庫保管場所を読み込んでいます...
          </div>
        ) : location ? (
          <div className="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle>
                  保管場所情報
                </CardTitle>
              </CardHeader>

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
                      value={location.name}
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
                      value={location.zipCode}
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
                      value={location.state}
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
                      value={location.city}
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
                      value={location.street}
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
                      value={location.street2}
                      onChange={(event) =>
                        handlers.onChangeStreet2(
                          event.target.value,
                        )
                      }
                      disabled={disabled}
                    />
                  </div>

                  <div>
                    <CardLabel htmlFor="location-country">
                      国
                    </CardLabel>

                    <CardInput
                      id="location-country"
                      value="日本"
                      disabled
                      readOnly
                    />
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>
                  管理情報
                </CardTitle>
              </CardHeader>

              <CardContent>
                <div className="space-y-4 text-sm">
                  <div>
                    <p className="text-xs text-slate-500">
                      登録日
                    </p>

                    <p className="mt-1 text-slate-900">
                      {createdAt || "-"}
                    </p>
                  </div>

                  <div>
                    <p className="text-xs text-slate-500">
                      最終更新日
                    </p>

                    <p className="mt-1 text-slate-900">
                      {updatedAt || "-"}
                    </p>
                  </div>
                </div>
              </CardContent>
            </Card>

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

            {vm.deleting && (
              <p className="text-sm text-slate-500">
                在庫保管場所を削除しています...
              </p>
            )}
          </div>
        ) : (
          <div className="rounded-lg border border-slate-200 bg-white px-5 py-12 text-center text-sm text-slate-500">
            在庫保管場所を表示できませんでした。
          </div>
        )}
      </div>
    </PageStyle>
  );
}