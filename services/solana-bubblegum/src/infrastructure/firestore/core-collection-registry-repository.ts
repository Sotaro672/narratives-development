// services/solana-bubblegum/src/infrastructure/firestore/core-collection-registry-repository.ts

import {
  Timestamp,
} from "@google-cloud/firestore";

import type {
  CoreCollectionRegistryPort,
  CoreCollectionRegistryRecord,
} from "../../application/ports/core-collection-registry-port.js";

import {
  firestore,
} from "./firestore-client.js";


type FirestoreCoreCollectionRecord = {
  tokenBlueprintId?: unknown;

  collectionAddress?: unknown;

  name?: unknown;

  metadataUri?: unknown;

  cluster?: unknown;

  txSignature?: unknown;

  createdAt?: unknown;

  updatedAt?: unknown;
};


function requiredString(
  field: string,
  value: unknown,
): string {
  if (
    typeof value !== "string" ||
    value.length === 0
  ) {
    throw new Error(
      `core_collection_registry: invalid ${field}`,
    );
  }

  return value;
}


function requiredDate(
  field: string,
  value: unknown,
): Date {
  if (
    value instanceof Timestamp
  ) {
    return value.toDate();
  }

  if (
    value instanceof Date
  ) {
    return value;
  }

  throw new Error(
    `core_collection_registry: invalid ${field}`,
  );
}


function fromFirestore(
  data: FirestoreCoreCollectionRecord,
): CoreCollectionRegistryRecord {
  return {
    tokenBlueprintId:
      requiredString(
        "tokenBlueprintId",
        data.tokenBlueprintId,
      ),

    collectionAddress:
      requiredString(
        "collectionAddress",
        data.collectionAddress,
      ),

    name:
      requiredString(
        "name",
        data.name,
      ),

    metadataUri:
      requiredString(
        "metadataUri",
        data.metadataUri,
      ),

    cluster:
      requiredString(
        "cluster",
        data.cluster,
      ),

    txSignature:
      requiredString(
        "txSignature",
        data.txSignature,
      ),

    createdAt:
      requiredDate(
        "createdAt",
        data.createdAt,
      ),

    updatedAt:
      requiredDate(
        "updatedAt",
        data.updatedAt,
      ),
  };
}


export class FirestoreCoreCollectionRegistryRepository
  implements CoreCollectionRegistryPort {
  async getByTokenBlueprintId(
    tokenBlueprintId: string,
  ): Promise<CoreCollectionRegistryRecord | null> {
    if (!tokenBlueprintId) {
      throw new Error(
        "core_collection_registry: tokenBlueprintId is required",
      );
    }


    const snapshot =
      await firestore
        .collection(
          "bubblegumCoreCollections",
        )
        .doc(
          tokenBlueprintId,
        )
        .get();


    if (!snapshot.exists) {
      return null;
    }


    return fromFirestore(
      snapshot.data() as FirestoreCoreCollectionRecord,
    );
  }


  async save(
    record: CoreCollectionRegistryRecord,
  ): Promise<void> {
    if (!record.tokenBlueprintId) {
      throw new Error(
        "core_collection_registry: tokenBlueprintId is required",
      );
    }


    if (!record.collectionAddress) {
      throw new Error(
        "core_collection_registry: collectionAddress is required",
      );
    }


    const ref =
      firestore
        .collection(
          "bubblegumCoreCollections",
        )
        .doc(
          record.tokenBlueprintId,
        );


    await firestore.runTransaction(
      async (transaction) => {
        const snapshot =
          await transaction.get(
            ref,
          );


        if (snapshot.exists) {
          const existing =
            fromFirestore(
              snapshot.data() as FirestoreCoreCollectionRecord,
            );


          if (
            existing.collectionAddress !==
            record.collectionAddress
          ) {
            throw new Error(
              [
                "core_collection_registry: collection conflict",
                `tokenBlueprintId=${record.tokenBlueprintId}`,
                `existing=${existing.collectionAddress}`,
                `new=${record.collectionAddress}`,
              ].join(
                " ",
              ),
            );
          }


          return;
        }


        transaction.create(
          ref,
          {
            tokenBlueprintId:
              record.tokenBlueprintId,

            collectionAddress:
              record.collectionAddress,

            name:
              record.name,

            metadataUri:
              record.metadataUri,

            cluster:
              record.cluster,

            txSignature:
              record.txSignature,

            createdAt:
              Timestamp.fromDate(
                record.createdAt,
              ),

            updatedAt:
              Timestamp.fromDate(
                record.updatedAt,
              ),
          },
        );
      },
    );
  }
}