// backend/cmd/seed_category/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"cloud.google.com/go/firestore"
)

const collectionName = "productBlueprintCategories"

type CategorySeed struct {
	ProductBlueprintCategoryPath []string `firestore:"productBlueprintCategoryPath"`
}

func main() {
	ctx := context.Background()

	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		projectID = os.Getenv("FIREBASE_PROJECT_ID")
	}
	if projectID == "" {
		projectID = os.Getenv("GCP_PROJECT_ID")
	}
	if projectID == "" {
		log.Fatal("GOOGLE_CLOUD_PROJECT or FIREBASE_PROJECT_ID or GCP_PROJECT_ID is required")
	}

	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("failed to create firestore client: %v", err)
	}

	defer client.Close()

	categories := buildCategories()

	batch := client.Batch()

	for _, category := range categories {
		documentID := categoryDocumentID(
			category.ProductBlueprintCategoryPath,
		)

		ref := client.Collection(collectionName).Doc(documentID)

		batch.Set(ref, category)
	}

	if _, err := batch.Commit(ctx); err != nil {
		log.Fatalf("failed to seed categories: %v", err)
	}

	fmt.Printf("seeded %d product blueprint categories into %s\n", len(categories), collectionName)
}

func categoryDocumentID(
	productBlueprintCategoryPath []string,
) string {
	return strings.Join(
		productBlueprintCategoryPath,
		".",
	)
}

func buildCategories() []CategorySeed {
	return []CategorySeed{
		// ------------------------------------------------------------
		// apparel
		// ------------------------------------------------------------
		category(
			"apparel",
		),
		category(
			"apparel",
			"tops",
		),
		category(
			"apparel",
			"bottoms",
		),
		category(
			"apparel",
			"outerwear",
		),
		category(
			"apparel",
			"dress",
		),
		category(
			"apparel",
			"shoes",
		),
		category(
			"apparel",
			"bag",
		),
		category(
			"apparel",
			"accessory",
		),

		// ------------------------------------------------------------
		// alcohol
		// ------------------------------------------------------------
		category(
			"alcohol",
		),
		category(
			"alcohol",
			"sake",
		),
		category(
			"alcohol",
			"wine",
		),
		category(
			"alcohol",
			"beer",
		),
		category(
			"alcohol",
			"whisky",
		),
		category(
			"alcohol",
			"shochu",
		),
		category(
			"alcohol",
			"spirits",
		),

		// ------------------------------------------------------------
		// cosmetics
		// ------------------------------------------------------------
		category(
			"cosmetics",
		),
		category(
			"cosmetics",
			"skincare",
		),
		category(
			"cosmetics",
			"makeup",
		),
		category(
			"cosmetics",
			"fragrance",
		),
		category(
			"cosmetics",
			"haircare",
		),
		category(
			"cosmetics",
			"bodycare",
		),

		// ------------------------------------------------------------
		// healthcare
		// ------------------------------------------------------------
		category(
			"healthcare",
		),
		category(
			"healthcare",
			"supplement",
		),
		category(
			"healthcare",
			"wellness",
		),
		category(
			"healthcare",
			"medical_device",
		),

		// ------------------------------------------------------------
		// other
		// ------------------------------------------------------------
		category(
			"other",
		),
		category(
			"other",
			"general",
		),
	}
}

func category(
	productBlueprintCategoryPath ...string,
) CategorySeed {
	return CategorySeed{
		ProductBlueprintCategoryPath: append(
			[]string(nil),
			productBlueprintCategoryPath...,
		),
	}
}
