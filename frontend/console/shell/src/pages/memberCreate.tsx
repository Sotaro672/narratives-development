// frontend/console/shell/src/pages/memberCreate.tsx

import * as React from "react";
import { useNavigate } from "react-router-dom";

import { BrandSelect } from "../features/member/presentation/components/BrandSelect";
import { PermissionSelect } from "../features/member/presentation/components/PermissionSelect";
import { useMemberCreate } from "../features/member/presentation/hooks/useMemberCreate";
import PageStyle from "../layout/PageStyle/PageStyle";
import { ErrorMessage } from "../shared/ui/error";
import { Input } from "../shared/ui/input";

import "../styles/member.css";

export default function MemberCreatePage() {
  const navigate = useNavigate();
  const formRef = React.useRef<HTMLFormElement>(null);

  const {
    email,
    category,
    setCategory,
    submitting,
    error,
    setEmail,
    handleSubmit,
    permissionCategories,
    permissionCategoryList,
    brandRows,
  } = useMemberCreate({
    onSuccess: () => navigate("/member"),
  });

  const handleBack = () => navigate(-1);
  const handleCreate = () => formRef.current?.requestSubmit();

  const [selectedPermIds, setSelectedPermIds] = React.useState<Set<string>>(
    new Set(),
  );

  const allPerms = React.useMemo(
    () => permissionCategories.flatMap((c) => c.permissions as any[]),
    [permissionCategories],
  );

  const selectedPerms = React.useMemo(
    () => allPerms.filter((p) => selectedPermIds.has(p.id)),
    [allPerms, selectedPermIds],
  );

  const [selectedBrandIds, setSelectedBrandIds] = React.useState<Set<string>>(
    new Set(),
  );

  const toggleBrand = (id: string, explicit?: boolean) => {
    setSelectedBrandIds((prev) => {
      const next = new Set(prev);
      const willCheck = explicit ?? !next.has(id);

      if (willCheck) {
        next.add(id);
      } else {
        next.delete(id);
      }

      return next;
    });
  };

  const onSubmit = (e: React.FormEvent) => {
    const permissionNames = selectedPerms.map((p: any) => p.name as string);
    const brandIdsArray = Array.from(selectedBrandIds);

    handleSubmit(e, {
      permissions: permissionNames,
      assignedBrandIds: brandIdsArray,
    });
  };

  return (
    <PageStyle
      title="メンバー追加"
      onBack={handleBack}
      onCreate={handleCreate}
    >
      <div className="member-create">
        {error && (
          <ErrorMessage className="member-create__error">
            エラー: {error}
          </ErrorMessage>
        )}

        <form
          ref={formRef}
          onSubmit={onSubmit}
          className="member-create__form"
          noValidate
        >
          <BrandSelect
            brandRows={brandRows}
            selectedBrandIds={selectedBrandIds}
            onToggleBrand={toggleBrand}
          />

          <div>
            <label className="card__label">
              メールアドレス（必須）
            </label>

            <Input
              type="email"
              required
              autoComplete="email"
              variant="default"
              className="card__input"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="taro@example.com"
              disabled={submitting}
            />
          </div>

          <PermissionSelect
            category={category}
            setCategory={setCategory}
            permissionCategories={permissionCategories}
            permissionCategoryList={permissionCategoryList}
            selectedPermIds={selectedPermIds}
            setSelectedPermIds={setSelectedPermIds}
          />
        </form>
      </div>
    </PageStyle>
  );
}