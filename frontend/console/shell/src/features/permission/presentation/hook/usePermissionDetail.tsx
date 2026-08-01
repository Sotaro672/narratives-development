// frontend/console/shell/src/features/permission/presentation/hook/usePermissionDetail.tsx

import {
  useCallback,
  useEffect,
  useState,
} from "react";

import {
  useNavigate,
  useParams,
} from "react-router-dom";

import type {
  Permission,
} from "../../../../shared/types/permission";

import {
  PermissionRepositoryHTTP,
} from "../../infrastructure/http/permissionRepositoryHTTP";

const permissionRepository =
  new PermissionRepositoryHTTP();

/**
 * 権限詳細ページ用のロジック。
 *
 * URLのpermissionIdを使用して、
 * BackendからPermissionを取得する。
 */
export function usePermissionDetail() {
  const navigate =
    useNavigate();

  const {
    permissionId,
  } = useParams<{
    permissionId: string;
  }>();

  const [
    permission,
    setPermission,
  ] = useState<Permission | null>(
    null,
  );

  const [
    loading,
    setLoading,
  ] = useState(true);

  const [
    error,
    setError,
  ] = useState<string | null>(
    null,
  );

  useEffect(
    () => {
      let cancelled =
        false;

      const loadPermission =
        async () => {
          if (!permissionId) {
            if (!cancelled) {
              setPermission(
                null,
              );

              setError(
                "権限IDが指定されていません。",
              );

              setLoading(
                false,
              );
            }

            return;
          }

          setLoading(
            true,
          );

          setError(
            null,
          );

          try {
            const result =
              await permissionRepository.getById(
                permissionId,
              );

            if (cancelled) {
              return;
            }

            setPermission(
              result,
            );
          } catch (
            cause: unknown
          ) {
            if (cancelled) {
              return;
            }

            setPermission(
              null,
            );

            setError(
              cause instanceof Error
                ? cause.message
                : "権限詳細の取得に失敗しました。",
            );
          } finally {
            if (!cancelled) {
              setLoading(
                false,
              );
            }
          }
        };

      void loadPermission();

      return () => {
        cancelled =
          true;
      };
    },
    [
      permissionId,
    ],
  );

  const handleBack =
    useCallback(
      () => {
        navigate(
          -1,
        );
      },
      [
        navigate,
      ],
    );

  const title =
    permission
      ? `権限詳細：${permission.name}`
      : "権限詳細";

  return {
    permission,
    loading,
    error,
    handleBack,
    title,
  };
}

export default usePermissionDetail;