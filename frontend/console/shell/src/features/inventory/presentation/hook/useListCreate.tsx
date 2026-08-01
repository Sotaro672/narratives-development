// frontend/console/shell/src/features/inventory/presentation/hook/useListCreate.tsx

import * as React from "react";
import {
  useNavigate,
  useParams,
  type NavigateFunction,
} from "react-router-dom";

import { usePriceCard } from "../../../list/presentation/hook/usePriceCard";
import { useAdminCard } from "../../../admin/presentation/hook/useAdminCard";
import {
  useAuthContext,
} from "../../../../auth/application/AuthContext";

import type { ListStatus } from "../../../../shared/types/list";

import {
  buildAfterCreatePath,
  buildBackPath,
  canFetchListCreate,
  createListWithImages,
  extractDisplayStrings,
  loadListCreateDTOFromParams,
  resolveListCreateParams,
  type ListCreateRouteParams,
  type PriceRow,
  type ResolvedListCreateParams,
} from "../../application/listCreate/listCreateService";

import type { ListCreateDTO } from "../../infrastructure/http/listCreateRepositoryHTTP.types";

type ImageInputRef =
  React.RefObject<HTMLInputElement | null>;

type AssigneeCandidate = {
  id: string;
  name: string;
};

export type UseListCreateResult = {
  title: string;
  onBack: () => void;
  onCreate: () => void;

  dto: ListCreateDTO | null;
  loadingDTO: boolean;
  dtoError: string;

  productBrandName: string;
  productName: string;
  tokenBrandName: string;
  tokenName: string;

  priceRows: PriceRow[];
  onChangePrice: (
    index: number,
    price: number | undefined,
  ) => void;
  priceCard:
    ReturnType<typeof usePriceCard>;

  listingTitle: string;
  setListingTitle:
    React.Dispatch<
      React.SetStateAction<string>
    >;
  description: string;
  setDescription:
    React.Dispatch<
      React.SetStateAction<string>
    >;

  images: File[];
  imagePreviewUrls: string[];
  mainImageIndex: number;
  setMainImageIndex:
    React.Dispatch<
      React.SetStateAction<number>
    >;
  imageInputRef: ImageInputRef;
  onAddImages: (
    files: FileList | null,
  ) => void;
  onRemoveImageAt: (
    index: number,
  ) => void;
  onClearImages: () => void;

  assigneeName: string;
  assigneeCandidates:
    AssigneeCandidate[];
  loadingMembers: boolean;
  handleSelectAssignee: (
    id: string,
  ) => void;

  status: ListStatus;
  setStatus:
    React.Dispatch<
      React.SetStateAction<ListStatus>
    >;
};

type UsePriceRowsResult = {
  priceRows: PriceRow[];
  setPriceRows:
    React.Dispatch<
      React.SetStateAction<PriceRow[]>
    >;
  initializedPriceRowsRef:
    React.MutableRefObject<boolean>;
  onChangePrice: (
    index: number,
    price: number | undefined,
  ) => void;
  priceCard:
    ReturnType<typeof usePriceCard>;
};

function getMemberUid(
  member: unknown,
): string {
  const target =
    member as any;

  return String(
    target?.uid ?? "",
  );
}

function getMemberDisplayName(
  member: unknown,
): string {
  const target =
    member as any;

  const fullName =
    String(
      target?.fullName ?? "",
    );

  if (fullName) {
    return fullName;
  }

  const nameParts = [
    target?.lastName,
    target?.firstName,
  ]
    .map((value) =>
      String(value ?? ""),
    )
    .filter(Boolean);

  const joinedName =
    nameParts.join(" ");

  if (joinedName) {
    return joinedName;
  }

  const email =
    String(
      target?.email ?? "",
    );

  if (email) {
    return email;
  }

  const uid =
    getMemberUid(member);

  if (uid) {
    return uid;
  }

  return String(
    target?.id ?? "",
  );
}

function buildAssigneeCandidates(
  rawCandidates: unknown,
): AssigneeCandidate[] {
  const rows =
    Array.isArray(rawCandidates)
      ? rawCandidates
      : [];

  return rows
    .map((rawCandidate) => {
      const candidate =
        rawCandidate as any;

      const id =
        String(
          candidate?.uid ??
            candidate?.id ??
            "",
        );

      if (!id) {
        return null;
      }

      const nameParts = [
        candidate?.lastName,
        candidate?.firstName,
      ]
        .map((value) =>
          String(value ?? ""),
        )
        .filter(Boolean);

      const joinedName =
        nameParts.join(" ");

      const name =
        String(
          candidate?.name ?? "",
        ) ||
        String(
          candidate?.fullName ?? "",
        ) ||
        joinedName ||
        String(
          candidate?.email ?? "",
        ) ||
        id;

      return {
        id,
        name,
      };
    })
    .filter(
      Boolean,
    ) as AssigneeCandidate[];
}

function dedupeFiles(
  previousFiles: File[],
  addedFiles: File[],
): File[] {
  const existingKeys =
    new Set(
      previousFiles.map(
        (file) =>
          `${file.name}__${file.size}__${file.lastModified}`,
      ),
    );

  const filteredFiles =
    addedFiles.filter(
      (file) =>
        !existingKeys.has(
          `${file.name}__${file.size}__${file.lastModified}`,
        ),
    );

  return [
    ...previousFiles,
    ...filteredFiles,
  ];
}

function filterImageFiles(
  files:
    | FileList
    | File[]
    | null
    | undefined,
): File[] {
  return Array.from(
    files ?? [],
  )
    .filter(Boolean)
    .filter((file) =>
      String(
        file.type || "",
      ).startsWith(
        "image/",
      ),
    ) as File[];
}

function useListCreateParamsAndTitle(): {
  resolvedParams:
    ResolvedListCreateParams;
  title: string;
} {
  const params =
    useParams<ListCreateRouteParams>();

  const resolvedParams =
    React.useMemo(
      () =>
        resolveListCreateParams(
          params,
        ),
      [params],
    );

  return {
    resolvedParams,
    title: "出品作成",
  };
}

function useListingStatus(): {
  status: ListStatus;
  setStatus:
    React.Dispatch<
      React.SetStateAction<ListStatus>
    >;
} {
  const [
    status,
    setStatus,
  ] =
    React.useState<ListStatus>(
      "listing",
    );

  return {
    status,
    setStatus,
  };
}

function useListingFields(): {
  listingTitle: string;
  setListingTitle:
    React.Dispatch<
      React.SetStateAction<string>
    >;
  description: string;
  setDescription:
    React.Dispatch<
      React.SetStateAction<string>
    >;
} {
  const [
    listingTitle,
    setListingTitle,
  ] =
    React.useState("");

  const [
    description,
    setDescription,
  ] =
    React.useState("");

  return {
    listingTitle,
    setListingTitle,
    description,
    setDescription,
  };
}

function useListingImages(): {
  images: File[];
  imagePreviewUrls: string[];
  mainImageIndex: number;
  setMainImageIndex:
    React.Dispatch<
      React.SetStateAction<number>
    >;
  imageInputRef: ImageInputRef;
  onSelectImages: (
    files: FileList | null,
  ) => void;
  removeImageAt: (
    index: number,
  ) => void;
  clearImages: () => void;
} {
  const [
    images,
    setImages,
  ] =
    React.useState<File[]>(
      [],
    );

  const [
    mainImageIndex,
    setMainImageIndex,
  ] =
    React.useState(0);

  const [
    imagePreviewUrls,
    setImagePreviewUrls,
  ] =
    React.useState<string[]>(
      [],
    );

  const imageInputRef =
    React.useRef<
      HTMLInputElement | null
    >(null);

  const appendImages =
    React.useCallback(
      (
        filesLike:
          | FileList
          | File[]
          | null,
      ) => {
        const files =
          filterImageFiles(
            filesLike,
          );

        if (
          files.length === 0
        ) {
          return;
        }

        setImages(
          (
            previousFiles,
          ) =>
            dedupeFiles(
              previousFiles,
              files,
            ),
        );
      },
      [],
    );

  const onSelectImages =
    React.useCallback(
      (
        files:
          FileList | null,
      ) => {
        appendImages(
          files,
        );
      },
      [
        appendImages,
      ],
    );

  const removeImageAt =
    React.useCallback(
      (
        index: number,
      ) => {
        setImages(
          (
            previousFiles,
          ) =>
            previousFiles.filter(
              (
                _,
                previousIndex,
              ) =>
                previousIndex !==
                index,
            ),
        );

        setMainImageIndex(
          (
            previousMainIndex,
          ) => {
            if (
              index ===
              previousMainIndex
            ) {
              return 0;
            }

            if (
              index <
              previousMainIndex
            ) {
              return Math.max(
                0,
                previousMainIndex -
                  1,
              );
            }

            return previousMainIndex;
          },
        );
      },
      [],
    );

  const clearImages =
    React.useCallback(
      () => {
        setImages([]);
        setMainImageIndex(
          0,
        );
      },
      [],
    );

  React.useEffect(
    () => {
      if (
        images.length === 0
      ) {
        setImagePreviewUrls(
          [],
        );

        return;
      }

      const urls =
        images.map(
          (file) =>
            URL.createObjectURL(
              file,
            ),
        );

      setImagePreviewUrls(
        urls,
      );

      return () => {
        urls.forEach(
          (url) => {
            URL.revokeObjectURL(
              url,
            );
          },
        );
      };
    },
    [
      images,
    ],
  );

  React.useEffect(
    () => {
      if (
        images.length === 0
      ) {
        if (
          mainImageIndex !==
          0
        ) {
          setMainImageIndex(
            0,
          );
        }

        return;
      }

      if (
        mainImageIndex <
          0 ||
        mainImageIndex >=
          images.length
      ) {
        setMainImageIndex(
          0,
        );
      }
    },
    [
      images.length,
      mainImageIndex,
    ],
  );

  return {
    images,
    imagePreviewUrls,
    mainImageIndex,
    setMainImageIndex,
    imageInputRef,
    onSelectImages,
    removeImageAt,
    clearImages,
  };
}

function usePriceRows():
  UsePriceRowsResult {
  const [
    priceRows,
    setPriceRows,
  ] =
    React.useState<
      PriceRow[]
    >([]);

  const initializedPriceRowsRef =
    React.useRef(
      false,
    );

  const onChangePrice =
    React.useCallback(
      (
        index: number,
        price:
          number |
          undefined,
      ) => {
        setPriceRows(
          (
            previousRows,
          ) => {
            const currentRow =
              previousRows[
                index
              ];

            if (
              !currentRow
            ) {
              return previousRows;
            }

            const nextRows = [
              ...previousRows,
            ];

            nextRows[
              index
            ] = {
              ...currentRow,
              price,
            };

            return nextRows;
          },
        );
      },
      [],
    );

  const priceCard =
    usePriceCard({
      title: "価格",
      rows:
        priceRows,
      mode: "edit",
      currencySymbol:
        "¥",
      onChangePrice,
    });

  return {
    priceRows,
    setPriceRows,
    initializedPriceRowsRef,
    onChangePrice,
    priceCard,
  };
}

function useListCreateNavigation(
  resolvedParams:
    ResolvedListCreateParams,
): {
  navigate:
    NavigateFunction;
  onBack: () => void;
} {
  const navigate =
    useNavigate();

  const onBack =
    React.useCallback(
      () => {
        navigate(
          buildBackPath(
            resolvedParams,
          ),
        );
      },
      [
        navigate,
        resolvedParams,
      ],
    );

  return {
    navigate,
    onBack,
  };
}

function useListCreateDTO(
  args: {
    resolvedParams:
      ResolvedListCreateParams;
    initializedPriceRowsRef:
      React.MutableRefObject<boolean>;
    setPriceRows:
      React.Dispatch<
        React.SetStateAction<
          PriceRow[]
        >
      >;
  },
): {
  dto:
    ListCreateDTO |
    null;
  loadingDTO: boolean;
  dtoError: string;
  productBrandName: string;
  productName: string;
  tokenBrandName: string;
  tokenName: string;
} {
  const {
    resolvedParams,
    initializedPriceRowsRef,
    setPriceRows,
  } = args;

  const [
    dto,
    setDTO,
  ] =
    React.useState<
      ListCreateDTO |
      null
    >(null);

  const [
    loadingDTO,
    setLoadingDTO,
  ] =
    React.useState(
      false,
    );

  const [
    dtoError,
    setDTOError,
  ] =
    React.useState(
      "",
    );

  React.useEffect(
    () => {
      let cancelled =
        false;

      const run =
        async () => {
          if (
            !canFetchListCreate(
              resolvedParams,
            )
          ) {
            return;
          }

          setLoadingDTO(
            true,
          );

          setDTOError(
            "",
          );

          try {
            const data =
              await loadListCreateDTOFromParams(
                resolvedParams,
              );

            if (
              cancelled
            ) {
              return;
            }

            setDTO(
              data,
            );

            if (
              !initializedPriceRowsRef.current
            ) {
              setPriceRows(
                data.priceRows,
              );

              initializedPriceRowsRef.current =
                true;
            }
          } catch (
            error
          ) {
            if (
              cancelled
            ) {
              return;
            }

            setDTOError(
              String(
                error instanceof
                  Error
                  ? error.message
                  : error,
              ),
            );
          } finally {
            if (
              !cancelled
            ) {
              setLoadingDTO(
                false,
              );
            }
          }
        };

      void run();

      return () => {
        cancelled =
          true;
      };
    },
    [
      resolvedParams,
      initializedPriceRowsRef,
      setPriceRows,
    ],
  );

  const {
    productBrandName,
    productName,
    tokenBrandName,
    tokenName,
  } =
    React.useMemo(
      () =>
        extractDisplayStrings(
          dto,
        ),
      [
        dto,
      ],
    );

  return {
    dto,
    loadingDTO,
    dtoError,
    productBrandName,
    productName,
    tokenBrandName,
    tokenName,
  };
}

function useCreateList(
  args: {
    navigate:
      NavigateFunction;
    resolvedParams:
      ResolvedListCreateParams;
    status:
      ListStatus;
    listingTitle:
      string;
    description:
      string;
    priceRows:
      PriceRow[];
    assigneeId:
      string |
      undefined;
    images:
      File[];
    mainImageIndex:
      number;
  },
): {
  onCreate:
    () =>
      Promise<void>;
} {
  const {
    navigate,
    resolvedParams,
    status,
    listingTitle,
    description,
    priceRows,
    assigneeId,
    images,
    mainImageIndex,
  } = args;

  const onCreate =
    React.useCallback(
      async () => {
        let imageUploadFailedMessage =
          "";

        try {
          const inventoryId =
            String(
              resolvedParams.inventoryId ??
                "",
            );

          await createListWithImages({
            params: {
              ...resolvedParams,
              inventoryId,
            },

            listingTitle,
            description,
            priceRows,
            status,
            assigneeId,
            images,
            mainImageIndex,

            onImageUploadFailed:
              (
                message,
              ) => {
                imageUploadFailedMessage =
                  message;
              },
          });

          if (
            imageUploadFailedMessage
          ) {
            alert(
              imageUploadFailedMessage,
            );
          } else {
            alert(
              "作成しました",
            );
          }

          navigate(
            buildAfterCreatePath(
              resolvedParams,
            ),
          );
        } catch (
          error
        ) {
          alert(
            String(
              error instanceof
                Error
                ? error.message
                : error,
            ),
          );
        }
      },
      [
        navigate,
        resolvedParams,
        status,
        listingTitle,
        description,
        priceRows,
        assigneeId,
        images,
        mainImageIndex,
      ],
    );

  return {
    onCreate,
  };
}

export function useListCreate():
  UseListCreateResult {
  const {
    resolvedParams,
    title,
  } =
    useListCreateParamsAndTitle();

  const {
    currentMember,
  } =
    useAuthContext();

  const {
    status,
    setStatus,
  } =
    useListingStatus();

  const {
    listingTitle,
    setListingTitle,
    description,
    setDescription,
  } =
    useListingFields();

  const {
    images,
    imagePreviewUrls,
    mainImageIndex,
    setMainImageIndex,
    imageInputRef,
    onSelectImages,
    removeImageAt,
    clearImages,
  } =
    useListingImages();

  const {
    priceRows,
    setPriceRows,
    initializedPriceRowsRef,
    onChangePrice,
    priceCard,
  } =
    usePriceRows();

  const {
    navigate,
    onBack,
  } =
    useListCreateNavigation(
      resolvedParams,
    );

  const {
    assigneeCandidates:
      rawAssigneeCandidates,
    loadingMembers,
  } =
    useAdminCard();

  const assigneeCandidates =
    React.useMemo(
      () =>
        buildAssigneeCandidates(
          rawAssigneeCandidates,
        ),
      [
        rawAssigneeCandidates,
      ],
    );

  const [
    assigneeId,
    setAssigneeId,
  ] =
    React.useState(
      "",
    );

  const [
    assigneeName,
    setAssigneeName,
  ] =
    React.useState(
      "",
    );

  React.useEffect(
    () => {
      if (
        !currentMember ||
        assigneeId
      ) {
        return;
      }

      const memberUid =
        getMemberUid(
          currentMember,
        );

      if (
        !memberUid
      ) {
        return;
      }

      setAssigneeId(
        memberUid,
      );

      setAssigneeName(
        getMemberDisplayName(
          currentMember,
        ),
      );
    },
    [
      currentMember,
      assigneeId,
    ],
  );

  const handleSelectAssignee =
    React.useCallback(
      (
        id: string,
      ) => {
        const nextId =
          String(
            id ?? "",
          );

        if (
          !nextId
        ) {
          return;
        }

        const matched =
          assigneeCandidates.find(
            (
              candidate,
            ) =>
              candidate.id ===
              nextId,
          );

        const nextName =
          matched?.name ??
          (
            getMemberUid(
              currentMember,
            ) === nextId
              ? getMemberDisplayName(
                  currentMember,
                )
              : nextId
          );

        setAssigneeId(
          nextId,
        );

        setAssigneeName(
          nextName,
        );
      },
      [
        assigneeCandidates,
        currentMember,
      ],
    );

  const {
    dto,
    loadingDTO,
    dtoError,
    productBrandName,
    productName,
    tokenBrandName,
    tokenName,
  } =
    useListCreateDTO({
      resolvedParams,
      initializedPriceRowsRef,
      setPriceRows,
    });

  const {
    onCreate,
  } =
    useCreateList({
      navigate,
      resolvedParams,
      status,
      listingTitle,
      description,
      priceRows,
      assigneeId,
      images,
      mainImageIndex,
    });

  return {
    title,
    onBack,
    onCreate,

    dto,
    loadingDTO,
    dtoError,

    productBrandName,
    productName,
    tokenBrandName,
    tokenName,

    priceRows,
    onChangePrice,
    priceCard,

    listingTitle,
    setListingTitle,
    description,
    setDescription,

    images,
    imagePreviewUrls,
    mainImageIndex,
    setMainImageIndex,
    imageInputRef,

    onAddImages:
      onSelectImages,

    onRemoveImageAt:
      removeImageAt,

    onClearImages:
      clearImages,

    assigneeName,
    assigneeCandidates,

    loadingMembers:
      Boolean(
        loadingMembers,
      ),

    handleSelectAssignee,

    status,
    setStatus,
  };
}