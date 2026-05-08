// frontend/inspector/lib/models/inspector_product_blueprint.dart

class InspectorModelRef {
  final String modelId;
  final int displayOrder;

  const InspectorModelRef({required this.modelId, required this.displayOrder});

  factory InspectorModelRef.fromJson(Map<String, dynamic> json) {
    return InspectorModelRef(
      modelId: (json['modelId'] ?? json['modelID'] ?? '') as String,
      displayOrder: (json['displayOrder'] is num)
          ? (json['displayOrder'] as num).toInt()
          : 0,
    );
  }
}

class InspectorProductBlueprint {
  final String id;
  final String productName;

  // ▼ 会社ID → 会社名
  final String companyName;

  // ▼ ブランドID → ブランド名
  final String brandName;

  final String itemType;
  final String fit;
  final String material;
  final double weight;
  final List<String> qualityAssurance;
  final String productIdTagType;
  final String assigneeId;

  // ✅ 追加: modelRefs（displayOrder含む）
  // backend が modelRefs を返す想定。未対応/欠落時は空配列。
  final List<InspectorModelRef> modelRefs;

  InspectorProductBlueprint({
    required this.id,
    required this.productName,
    required this.companyName,
    required this.brandName,
    required this.itemType,
    required this.fit,
    required this.material,
    required this.weight,
    required this.qualityAssurance,
    required this.productIdTagType,
    required this.assigneeId,
    required this.modelRefs,
  });

  factory InspectorProductBlueprint.fromJson(Map<String, dynamic> json) {
    // バックエンド側が companyName / brandName を返す前提。
    // もしまだ companyId / brandId しか無い場合はフォールバックする。
    final companyName =
        (json['companyName'] ?? json['companyId'] ?? '') as String;
    final brandName = (json['brandName'] ?? json['brandId'] ?? '') as String;

    // ✅ modelRefs: [{modelId, displayOrder}] を受け取る
    final rawModelRefs = json['modelRefs'];
    final modelRefs = (rawModelRefs is List)
        ? rawModelRefs
              .whereType<Map<String, dynamic>>()
              .map(InspectorModelRef.fromJson)
              .where((r) => r.modelId.isNotEmpty)
              .toList()
        : const <InspectorModelRef>[];

    // ✅ 念のため displayOrder 昇順に整列（0は末尾扱い）
    final sortedModelRefs = [...modelRefs]
      ..sort((a, b) {
        final ai = a.displayOrder == 0 ? 1 << 30 : a.displayOrder;
        final bi = b.displayOrder == 0 ? 1 << 30 : b.displayOrder;
        if (ai != bi) return ai.compareTo(bi);
        return a.modelId.compareTo(b.modelId);
      });

    return InspectorProductBlueprint(
      id: (json['id'] ?? '') as String,
      productName: (json['productName'] ?? '') as String,
      companyName: companyName,
      brandName: brandName,
      itemType: (json['itemType'] ?? '') as String,
      fit: (json['fit'] ?? '') as String,
      material: (json['material'] ?? '') as String,
      weight: (json['weight'] is num)
          ? (json['weight'] as num).toDouble()
          : 0.0,
      qualityAssurance:
          (json['qualityAssurance'] as List<dynamic>?)
              ?.map((e) => e.toString())
              .toList() ??
          const [],
      productIdTagType: (json['productIdTagType'] ?? '') as String,
      assigneeId: (json['assigneeId'] ?? '') as String,
      modelRefs: sortedModelRefs,
    );
  }
}
