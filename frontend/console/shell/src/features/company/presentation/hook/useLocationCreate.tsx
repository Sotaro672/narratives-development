// frontend/console/shell/src/features/company/presentation/hook/useLocationCreate.tsx

import {
  useCallback,
  useState,
} from "react";
import { useNavigate } from "react-router-dom";

import {
  createCompanyShippingAddress,
} from "../../application/companyDetailService";

type LocationCreateFieldErrors = {
  name: string | null;
  zipCode: string | null;
  state: string | null;
  city: string | null;
  street: string | null;
};

type LocationCreateForm = {
  name: string;
  zipCode: string;
  state: string;
  city: string;
  street: string;
  street2: string;
};

export type UseLocationCreateResult = {
  vm: {
    name: string;
    zipCode: string;
    state: string;
    city: string;
    street: string;
    street2: string;

    nameError: string | null;
    zipCodeError: string | null;
    stateError: string | null;
    cityError: string | null;
    streetError: string | null;

    saving: boolean;
    error: string | null;
  };

  handlers: {
    onChangeName: (
      value: string,
    ) => void;

    onChangeZipCode: (
      value: string,
    ) => void;

    onChangeState: (
      value: string,
    ) => void;

    onChangeCity: (
      value: string,
    ) => void;

    onChangeStreet: (
      value: string,
    ) => void;

    onChangeStreet2: (
      value: string,
    ) => void;

    onBack: () => void;
    onSave: () => Promise<void>;
  };
};

const emptyForm: LocationCreateForm = {
  name: "",
  zipCode: "",
  state: "",
  city: "",
  street: "",
  street2: "",
};

const emptyFieldErrors: LocationCreateFieldErrors = {
  name: null,
  zipCode: null,
  state: null,
  city: null,
  street: null,
};

function getErrorMessage(
  error: unknown,
): string {
  if (
    error instanceof Error &&
    error.message
  ) {
    return error.message;
  }

  return "在庫保管場所の登録に失敗しました。";
}

function validateForm(
  form: LocationCreateForm,
): LocationCreateFieldErrors {
  const errors: LocationCreateFieldErrors = {
    ...emptyFieldErrors,
  };

  if (!form.name) {
    errors.name =
      "保管場所名を入力してください。";
  } else if (
    form.name.length > 100
  ) {
    errors.name =
      "保管場所名は100文字以内で入力してください。";
  }

  if (!form.zipCode) {
    errors.zipCode =
      "郵便番号を入力してください。";
  } else if (
    !/^[0-9]{3}-?[0-9]{4}$/.test(
      form.zipCode,
    )
  ) {
    errors.zipCode =
      "郵便番号は123-4567または1234567の形式で入力してください。";
  }

  if (!form.state) {
    errors.state =
      "都道府県を入力してください。";
  } else if (
    form.state.length > 100
  ) {
    errors.state =
      "都道府県は100文字以内で入力してください。";
  }

  if (!form.city) {
    errors.city =
      "市区町村を入力してください。";
  } else if (
    form.city.length > 100
  ) {
    errors.city =
      "市区町村は100文字以内で入力してください。";
  }

  if (!form.street) {
    errors.street =
      "住所を入力してください。";
  } else if (
    form.street.length > 200
  ) {
    errors.street =
      "住所は200文字以内で入力してください。";
  }

  return errors;
}

function hasFieldError(
  errors: LocationCreateFieldErrors,
): boolean {
  return Object.values(
    errors,
  ).some(
    (value) =>
      value !== null,
  );
}

export function useLocationCreate(): UseLocationCreateResult {
  const navigate = useNavigate();

  const [
    form,
    setForm,
  ] = useState<LocationCreateForm>({
    ...emptyForm,
  });

  const [
    fieldErrors,
    setFieldErrors,
  ] = useState<LocationCreateFieldErrors>({
    ...emptyFieldErrors,
  });

  const [
    saving,
    setSaving,
  ] = useState(false);

  const [
    error,
    setError,
  ] = useState<string | null>(
    null,
  );

  const clearFieldError =
    useCallback(
      (
        field:
          keyof LocationCreateFieldErrors,
      ) => {
        setFieldErrors(
          (current) => ({
            ...current,
            [field]: null,
          }),
        );
      },
      [],
    );

  const onChangeName =
    useCallback(
      (
        value: string,
      ) => {
        setForm(
          (current) => ({
            ...current,
            name: value,
          }),
        );

        clearFieldError(
          "name",
        );

        setError(null);
      },
      [
        clearFieldError,
      ],
    );

  const onChangeZipCode =
    useCallback(
      (
        value: string,
      ) => {
        setForm(
          (current) => ({
            ...current,
            zipCode: value,
          }),
        );

        clearFieldError(
          "zipCode",
        );

        setError(null);
      },
      [
        clearFieldError,
      ],
    );

  const onChangeState =
    useCallback(
      (
        value: string,
      ) => {
        setForm(
          (current) => ({
            ...current,
            state: value,
          }),
        );

        clearFieldError(
          "state",
        );

        setError(null);
      },
      [
        clearFieldError,
      ],
    );

  const onChangeCity =
    useCallback(
      (
        value: string,
      ) => {
        setForm(
          (current) => ({
            ...current,
            city: value,
          }),
        );

        clearFieldError(
          "city",
        );

        setError(null);
      },
      [
        clearFieldError,
      ],
    );

  const onChangeStreet =
    useCallback(
      (
        value: string,
      ) => {
        setForm(
          (current) => ({
            ...current,
            street: value,
          }),
        );

        clearFieldError(
          "street",
        );

        setError(null);
      },
      [
        clearFieldError,
      ],
    );

  const onChangeStreet2 =
    useCallback(
      (
        value: string,
      ) => {
        setForm(
          (current) => ({
            ...current,
            street2: value,
          }),
        );

        setError(null);
      },
      [],
    );

  const onBack =
    useCallback(() => {
      navigate(-1);
    }, [
      navigate,
    ]);

  const onSave =
    useCallback(
      async (): Promise<void> => {
        if (saving) {
          return;
        }

        const nextFieldErrors =
          validateForm(
            form,
          );

        setFieldErrors(
          nextFieldErrors,
        );

        if (
          hasFieldError(
            nextFieldErrors,
          )
        ) {
          setError(
            "入力内容を確認してください。",
          );

          return;
        }

        setSaving(true);
        setError(null);

        try {
          const created =
            await createCompanyShippingAddress({
              name:
                form.name,
              zipCode:
                form.zipCode,
              state:
                form.state,
              city:
                form.city,
              street:
                form.street,
              street2:
                form.street2,
              country:
                "JP",
            });

          if (!created.id) {
            throw new Error(
              "在庫保管場所IDを取得できませんでした。",
            );
          }

          navigate(
            `/stockLocation/${encodeURIComponent(
              created.id,
            )}`,
            {
              replace: true,
            },
          );
        } catch (
          saveError: unknown
        ) {
          setError(
            getErrorMessage(
              saveError,
            ),
          );
        } finally {
          setSaving(false);
        }
      },
      [
        form,
        saving,
        navigate,
      ],
    );

  return {
    vm: {
      name:
        form.name,
      zipCode:
        form.zipCode,
      state:
        form.state,
      city:
        form.city,
      street:
        form.street,
      street2:
        form.street2,

      nameError:
        fieldErrors.name,
      zipCodeError:
        fieldErrors.zipCode,
      stateError:
        fieldErrors.state,
      cityError:
        fieldErrors.city,
      streetError:
        fieldErrors.street,

      saving,
      error,
    },

    handlers: {
      onChangeName,
      onChangeZipCode,
      onChangeState,
      onChangeCity,
      onChangeStreet,
      onChangeStreet2,

      onBack,
      onSave,
    },
  };
}

export default useLocationCreate;