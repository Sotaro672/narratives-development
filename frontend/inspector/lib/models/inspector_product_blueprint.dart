// frontend/inspector/lib/models/inspector_product_blueprint.dart
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
  });

  factory InspectorProductBlueprint.fromJson(Map<String, dynamic> json) {
    // バックエンド側が companyName / brandName を返す前提。
    // もしまだ companyId / brandId しか無い場合はフォールバックする。
    final companyName =
        (json['companyName'] ?? json['companyId'] ?? '') as String;
    final brandName = (json['brandName'] ?? json['brandId'] ?? '') as String;

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
    );
  }
}
