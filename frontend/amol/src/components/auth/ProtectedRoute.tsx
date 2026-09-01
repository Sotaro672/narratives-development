// frontend/amol/src/components/auth/ProtectedRoute.tsx

import {
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import {
  Navigate,
  useLocation,
} from "react-router-dom";
import {
  getAuth,
  onAuthStateChanged,
  type User,
} from "firebase/auth";

import {
  buildApiUrl,
  getApiBaseUrl,
} from "../../lib/apiBaseUrl";

type ProtectedRouteProps = {
  children: ReactNode;
};

type AvatarSetupStatus = {
  hasAvatar: boolean;
};

type AuthState = {
  user: User | null;
  loading: boolean;
};

type AvatarStatusState = {
  status: AvatarSetupStatus | null;
  loading: boolean;
  checked: boolean;
};

function readBoolean(
  value: unknown,
): boolean {
  if (typeof value === "boolean") {
    return value;
  }

  if (typeof value === "string") {
    return value === "true";
  }

  if (typeof value === "number") {
    return value !== 0;
  }

  return false;
}

async function fetchAvatarSetupStatus(
  user: User,
  backendUrl: string,
): Promise<AvatarSetupStatus | null> {
  if (!backendUrl) {
    return null;
  }

  const url = buildApiUrl(
    backendUrl,
    "/mall/me/setup-status",
  );

  async function getToken(
    forceRefresh: boolean,
  ): Promise<string | null> {
    const token =
      await user.getIdToken(forceRefresh);

    return token || null;
  }

  let token = await getToken(false);

  if (!token) {
    return null;
  }

  const headers: HeadersInit = {
    "Content-Type": "application/json",
    Accept: "application/json",
    Authorization: `Bearer ${token}`,
  };

  let response = await fetch(url, {
    method: "GET",
    headers,
  });

  if (
    response.status === 401 ||
    response.status === 403
  ) {
    token = await getToken(true);

    if (!token) {
      return null;
    }

    response = await fetch(url, {
      method: "GET",
      headers: {
        ...headers,
        Authorization: `Bearer ${token}`,
      },
    });
  }

  if (!response.ok) {
    return null;
  }

  const body = (await response
    .json()
    .catch(() => null)) as
    | {
        data?: {
          hasAvatar?: unknown;
          setupCompleted?: unknown;
        };
        hasAvatar?: unknown;
        setupCompleted?: unknown;
      }
    | null;

  if (!body) {
    return null;
  }

  const data = body.data ?? body;

  return {
    hasAvatar:
      readBoolean(data.hasAvatar) ||
      readBoolean(data.setupCompleted),
  };
}

export default function ProtectedRoute({
  children,
}: ProtectedRouteProps) {
  const location = useLocation();

  const backendUrl = useMemo(
    () => getApiBaseUrl(),
    [],
  );

  const [authState, setAuthState] =
    useState<AuthState>({
      user: null,
      loading: true,
    });

  const [avatarStatus, setAvatarStatus] =
    useState<AvatarStatusState>({
      status: null,
      loading: false,
      checked: false,
    });

  const isAvatarPage =
    location.pathname === "/avatar";

  const redirectPath =
    location.pathname +
    location.search +
    location.hash;

  useEffect(() => {
    const auth = getAuth();

    const unsubscribe =
      onAuthStateChanged(
        auth,
        (currentUser) => {
          setAuthState({
            user: currentUser,
            loading: false,
          });

          setAvatarStatus({
            status: null,
            loading: false,
            checked: false,
          });
        },
      );

    return () => unsubscribe();
  }, []);

  useEffect(() => {
    let cancelled = false;

    async function run() {
      const user = authState.user;

      if (!user) {
        setAvatarStatus({
          status: null,
          loading: false,
          checked: true,
        });

        return;
      }

      if (isAvatarPage) {
        setAvatarStatus({
          status: {
            hasAvatar: true,
          },
          loading: false,
          checked: true,
        });

        return;
      }

      setAvatarStatus({
        status: null,
        loading: true,
        checked: false,
      });

      const status =
        await fetchAvatarSetupStatus(
          user,
          backendUrl,
        );

      if (cancelled) {
        return;
      }

      setAvatarStatus({
        status,
        loading: false,
        checked: true,
      });
    }

    if (!authState.loading) {
      void run();
    }

    return () => {
      cancelled = true;
    };
  }, [
    authState.loading,
    authState.user,
    backendUrl,
    isAvatarPage,
  ]);

  if (authState.loading) {
    return null;
  }

  if (!authState.user) {
    return (
      <Navigate
        to={{
          pathname: "/signin",
          search: `?redirect=${encodeURIComponent(
            redirectPath,
          )}`,
        }}
        replace
      />
    );
  }

  if (
    avatarStatus.loading ||
    !avatarStatus.checked
  ) {
    return null;
  }

  if (
    !isAvatarPage &&
    !avatarStatus.status?.hasAvatar
  ) {
    return (
      <Navigate
        to="/avatar"
        replace
      />
    );
  }

  return <>{children}</>;
}