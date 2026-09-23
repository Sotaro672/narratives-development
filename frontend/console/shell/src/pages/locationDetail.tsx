// frontend/console/shell/src/pages/locationDetail.tsx

import { useCallback, useState } from "react";
import { useParams } from "react-router-dom";

import AdminCard from "../features/admin/presentation/components/AdminCard";
import LocationFormFields from "../features/company/presentation/components/LocationFormFields";
import { useLocationDetail } from "../features/company/presentation/hook/useLocationDetail";
import PageStyle from "../layout/PageStyle/PageStyle";
import { Card, CardContent } from "../shared/ui/card";
import Empty from "../shared/ui/empty";
import { ErrorMessage } from "../shared/ui/error";
import Stack from "../shared/ui/stack";
import Text from "../shared/ui/text";
import { safeDateTimeLabelJa } from "../shared/util/dateJa";

import "../styles/location.css";

export default function LocationDetail() {
  const { locationId } = useParams<{
    locationId: string;
  }>();

  const { vm, handlers } = useLocationDetail(locationId);
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

  const disabled = !isEditing || vm.saving || vm.deleting;

  const createdAt = location
    ? safeDateTimeLabelJa(location.createdAt, "")
    : "";

  const updatedAt = location
    ? safeDateTimeLabelJa(location.updatedAt, "")
    : "";

  const left = (
    <Stack gap="lg">
      {vm.loading ? (
        <Card>
          <Empty
            compact
            description="在庫保管場所を読み込んでいます..."
          />
        </Card>
      ) : location ? (
        <>
          <Card>
            <CardContent>
              <LocationFormFields
                value={location}
                errors={{
                  name: vm.nameError,
                  zipCode: vm.zipCodeError,
                  state: vm.stateError,
                  city: vm.cityError,
                  street: vm.streetError,
                }}
                disabled={disabled}
                onChangeName={handlers.onChangeName}
                onChangeZipCode={handlers.onChangeZipCode}
                onChangeState={handlers.onChangeState}
                onChangeCity={handlers.onChangeCity}
                onChangeStreet={handlers.onChangeStreet}
                onChangeStreet2={handlers.onChangeStreet2}
              />
            </CardContent>
          </Card>

          {vm.error && (
            <ErrorMessage variant="panel">
              {vm.error}
            </ErrorMessage>
          )}

          {vm.deleting && (
            <Text as="p" size="sm" tone="muted">
              在庫保管場所を削除しています...
            </Text>
          )}
        </>
      ) : (
        <Card>
          <Empty
            compact
            description="在庫保管場所を表示できませんでした。"
          />
        </Card>
      )}
    </Stack>
  );

  const right = (
    <div className="page-column">
      {vm.loading ? (
        <Card>
          <Empty
            compact
            description="管理情報を読み込んでいます..."
          />
        </Card>
      ) : location ? (
        <AdminCard
          title="管理情報"
          showAssignee={false}
          createdByName={location.createdByName}
          createdAt={createdAt}
          updatedByName={location.updatedByName}
          updatedAt={updatedAt}
          mode="view"
        />
      ) : null}
    </div>
  );

  return (
    <PageStyle
      layout="grid-2"
      title="在庫保管場所詳細"
      onBack={handlers.onBack}
      onEdit={!isEditing && location ? handleEdit : undefined}
      onDelete={isEditing ? handleDelete : undefined}
      onCancel={isEditing ? handleCancel : undefined}
      onSave={isEditing ? handleSave : undefined}
      isSaving={vm.saving}
    >
      {[left, right]}
    </PageStyle>
  );
}