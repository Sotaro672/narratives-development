// frontend/inspector/lib/models/inspector_product_blueprint.dart

class InspectorModelRef {
  final String modelId;
  final int displayOrder;

  const InspectorModelRef({required this.modelId, required this.displayOrder});

  factory InspectorModelRef.fromJson(Map<String, dynamic> json) {
    return InspectorModelRef(
      modelId: (json['modelId'] ?? json['modelID'] ?? '').toString(),
      displayOrder: json['displayOrder'] is num
          ? (json['displayOrder'] as num).toInt()
          : 0,
    );
  }
}

class InspectorProductBlueprint {
  final String id;
  final String productName;
  final String companyName;
  final String brandName;

  final String fit;
  final String material;
  final double? weight;
  final List<String> qualityAssurance;
  final String productIdTagType;
  final String assigneeId;

  final List<InspectorModelRef> modelRefs;

  InspectorProductBlueprint({
    required this.id,
    required this.productName,
    required this.companyName,
    required this.brandName,
    required this.fit,
    required this.material,
    required this.weight,
    required this.qualityAssurance,
    required this.productIdTagType,
    required this.assigneeId,
    required this.modelRefs,
  });

  factory InspectorProductBlueprint.fromJson(Map<String, dynamic> json) {
    final companyName = (json['companyName'] ?? json['companyId'] ?? '')
        .toString();
    final brandName = (json['brandName'] ?? json['brandId'] ?? '').toString();

    final rawCategoryFields = json['categoryFields'];
    final categoryFields = rawCategoryFields is Map<String, dynamic>
        ? rawCategoryFields
        : const <String, dynamic>{};

    double? parseWeight(dynamic value) {
      if (value == null) {
        return null;
      }

      if (value is num) {
        return value.toDouble();
      }

      return double.tryParse(value.toString());
    }

    final rawWashTags = categoryFields['washTags'];
    final qualityAssurance = rawWashTags is List
        ? rawWashTags.map((e) => e.toString()).toList()
        : const <String>[];

    final rawModelRefs = json['modelRefs'];
    final modelRefs = rawModelRefs is List
        ? rawModelRefs
              .whereType<Map<String, dynamic>>()
              .map(InspectorModelRef.fromJson)
              .where((ref) => ref.modelId.isNotEmpty)
              .toList()
        : const <InspectorModelRef>[];

    final sortedModelRefs = [...modelRefs]
      ..sort((a, b) {
        final aOrder = a.displayOrder == 0 ? 1 << 30 : a.displayOrder;
        final bOrder = b.displayOrder == 0 ? 1 << 30 : b.displayOrder;

        if (aOrder != bOrder) {
          return aOrder.compareTo(bOrder);
        }

        return a.modelId.compareTo(b.modelId);
      });

    return InspectorProductBlueprint(
      id: (json['id'] ?? '').toString(),
      productName: (json['productName'] ?? '').toString(),
      companyName: companyName,
      brandName: brandName,
      fit: (categoryFields['fit'] ?? '').toString(),
      material: (categoryFields['material'] ?? '').toString(),
      weight: parseWeight(categoryFields['weight']),
      qualityAssurance: qualityAssurance,
      productIdTagType: (json['productIdTagType'] ?? '').toString(),
      assigneeId: (json['assigneeId'] ?? '').toString(),
      modelRefs: sortedModelRefs,
    );
  }
}
