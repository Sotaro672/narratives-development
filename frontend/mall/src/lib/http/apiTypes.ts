// frontend/amol/src/lib/http/apiTypes.ts

export type HttpAuthMode =
  | "none"
  | "optional"
  | "required";

export type ApiQueryValue =
  | string
  | number
  | boolean
  | null
  | undefined;

export type ApiQueryParams =
  Record<
    string,
    ApiQueryValue
  >;

export type ApiErrorMessages = {
  requestErrorMessage: string;
  nonJsonErrorMessage: string;
  invalidJsonErrorMessage?: string;
};

export type ApiRequestOptions =
  Omit<
    RequestInit,
    "body"
  > & {
    auth?: HttpAuthMode;
    query?: ApiQueryParams;
    json?: unknown;
  };

export type ApiJsonRequestOptions<T> =
  ApiRequestOptions & {
    unwrapData?: boolean;
    fallbackValue?: T;
    messages?: Partial<
      ApiErrorMessages
    >;
  };