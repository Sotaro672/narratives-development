export type MarketResaleConditionImage = {
  id: string;
  resaleId?: string;
  url: string;
  objectPath?: string;
  fileName?: string;
  fileSize?: number;
  mimeType?: string;
  type?: string;
  displayOrder?: number;
  createdAt?: string;
  createdBy?: string;
  updatedAt?: string | null;
  updatedBy?: string | null;
};

export type MarketResaleConditionImagesResponse =
  | MarketResaleConditionImage[]
  | {
      data?: MarketResaleConditionImage[];
      items?: MarketResaleConditionImage[];
    };