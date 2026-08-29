// frontend/amol/src/features/resale/presentation/types/resaleDetailPageTypes.ts

import type { ChangeEvent, RefObject } from "react";

import type { MediaGalleryItem } from "../../../../components/ui/MediaGallery";
import type { MediaUploaderItem } from "../../../../components/ui/MediaUploader";

import type {
  ResaleModelDisplay,
} from "../../../shared/presentation/utils/resaleModelDisplay";
import type {
  ResaleCondition,
  ResaleConditionImage,
  ResaleEditableStatus,
  ResaleListing,
} from "../../../shared/types/resale";

export type ResaleDetailConditionMediaItem =
  Omit<MediaUploaderItem, "type"> & {
    type: "image";
    source: "existing" | "new";
    file?: File;
    image?: ResaleConditionImage;
  };

export type ResaleListingTargetSummary = {
  tokenIconUrl: string;
  tokenName: string;
  brandName: string;
  productName: string;
};

export type ResaleDetailReadonlyInfoProps = {
  galleryItems: MediaGalleryItem[];
  activeGalleryIndex: number;
  priceLabel: string;
  conditionLabel: string;
  statusLabel: string;
  createdAtLabel: string;
  updatedAtLabel: string;
  description: string;
  onPrevGalleryItem: () => void;
  onNextGalleryItem: () => void;
  onSelectGalleryItem: (index: number) => void;
};

export type ResaleDetailEditFormProps = {
  priceValue: string;
  condition: ResaleCondition;
  status: ResaleEditableStatus;
  description: string;
  saving: boolean;
  createdAtLabel: string;
  updatedAtLabel: string;
  conditionMediaItems: ResaleDetailConditionMediaItem[];
  conditionMediaCurrentIndex: number;
  conditionMediaInputRef: RefObject<HTMLInputElement>;
  conditionMediaCarouselRef: RefObject<HTMLDivElement>;
  onPriceChange: (value: string) => void;
  onConditionChange: (value: ResaleCondition) => void;
  onStatusChange: (value: ResaleEditableStatus) => void;
  onDescriptionChange: (value: string) => void;
  onConditionMediaSelected: (event: ChangeEvent<HTMLInputElement>) => void;
  onRemoveConditionMedia: (id: string) => void;
  onConditionMediaCarouselScroll: () => void;
  onMoveToConditionMediaSlide: (index: number) => void;
};

export type ResaleDetailActionFooterProps = {
  variant: "action";
  buttonLabel: string;
  disabled?: boolean;
  onButtonClick: () => void | Promise<void>;
};

export type ResaleDetailTripleActionFooterProps = {
  variant: "tripleAction";
  leftButtonLabel: string;
  centerButtonLabel: string;
  rightButtonLabel: string;
  leftButtonDisabled?: boolean;
  centerButtonDisabled?: boolean;
  rightButtonDisabled?: boolean;
  onLeftButtonClick: () => void | Promise<void>;
  onCenterButtonClick: () => void | Promise<void>;
  onRightButtonClick: () => void | Promise<void>;
};

export type ResaleDetailFooterProps =
  | ResaleDetailActionFooterProps
  | ResaleDetailTripleActionFooterProps;

export type ResaleDetailPageViewModel = {
  title: string;
  footerProps?: ResaleDetailFooterProps;
  loading: boolean;
  item: ResaleListing | null;
  isEditing: boolean;
  isSold: boolean;
  errorMessage: string;
  saveMessage: string;
  listingTarget: ResaleListingTargetSummary;
  model: ResaleModelDisplay;
  readonlyInfoProps: ResaleDetailReadonlyInfoProps;
  editFormProps: ResaleDetailEditFormProps;
  handleBack: () => void;
  handleReload: () => Promise<void>;
  handleBackToWallet: () => void;
};