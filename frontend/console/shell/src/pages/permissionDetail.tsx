// frontend/console/shell/src/pages/permissionDetail.tsx

import PageStyle from "../layout/PageStyle/PageStyle";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  CardLabel,
  CardReadonly,
} from "../shared/ui/card";
import { ErrorMessage } from "../shared/ui/error";

import { usePermissionDetail } from "../features/permission/presentation/hook/usePermissionDetail";

import "../styles/permission.css";

export default function PermissionDetail() {
  const {
    permission,
    loading,
    error,
    title,
    handleBack,
  } = usePermissionDetail();

  if (loading) {
    return (
      <PageStyle layout="single" title={title} onBack={handleBack}>
        <div className="permission-detail__message">
          読み込み中...
        </div>
      </PageStyle>
    );
  }

  if (error) {
    return (
      <PageStyle layout="single" title={title} onBack={handleBack}>
        <ErrorMessage>
          {error}
        </ErrorMessage>
      </PageStyle>
    );
  }

  if (!permission) {
    return (
      <PageStyle layout="single" title={title} onBack={handleBack}>
        <div className="permission-detail__message">
          権限情報が見つかりません。
        </div>
      </PageStyle>
    );
  }

  return (
    <PageStyle layout="single" title={title} onBack={handleBack}>
      <div className="permission-detail">
        <Card>
          <CardHeader>
            <CardTitle>基本情報</CardTitle>
          </CardHeader>

          <CardContent>
            <CardLabel>権限ID</CardLabel>
            <CardReadonly>{permission.id}</CardReadonly>

            <CardLabel>権限名</CardLabel>
            <CardReadonly>{permission.name}</CardReadonly>

            <CardLabel>カテゴリ</CardLabel>
            <CardReadonly>{permission.category}</CardReadonly>

            <CardLabel>説明</CardLabel>
            <div className="permission-detail__description">
              {permission.description}
            </div>
          </CardContent>
        </Card>
      </div>
    </PageStyle>
  );
}