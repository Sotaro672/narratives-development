// services/solana-bubblegum/src/application/ports/core-collection-registry-port.ts

export type CoreCollectionRegistryRecord = {
  tokenBlueprintId: string;

  collectionAddress: string;

  name: string;

  metadataUri: string;

  cluster: string;

  txSignature: string;

  createdAt: Date;

  updatedAt: Date;
};


export interface CoreCollectionRegistryPort {
  getByTokenBlueprintId(
    tokenBlueprintId: string,
  ): Promise<CoreCollectionRegistryRecord | null>;

  save(
    record: CoreCollectionRegistryRecord,
  ): Promise<void>;
}