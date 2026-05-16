//frontend\console\list\src\infrastructure\dto\listImageDto.ts
export type ListImageDTO = {
  id: string;

  url: string;
  objectPath: string;

  fileName?: string;
  contentType?: string;
  size: number;

  displayOrder: number;

  createdBy?: string;
  createdAt?: string;

  updatedBy?: string;
  updatedAt?: string;
};

export type SaveListImageFromFirebaseStorageInput = {
  listId: string;

  id: string;
  url: string;
  objectPath: string;

  size: number;
  displayOrder: number;

  fileName?: string;
  contentType?: string;

  createdBy?: string;
  createdAt?: string;
};