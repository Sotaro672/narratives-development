// frontend/inspector/lib/models/inspector_inspection_record.dart

class InspectorInspectionRecord {
  final String productId;

  /// InspectionItem の modelId
  final String? modelId;

  /// backend で解決された modelNumber
  final String? modelNumber;

  /// ✅ 追加: ProductBlueprint の modelRefs.displayOrder
  /// inspections 一覧を並べ替えるために使用
  final int? displayOrder;

  final String? inspectionResult;

  /// backend で名前解決された inspectedBy（表示名）
  final String? inspectedBy;

  final DateTime? inspectedAt;

  InspectorInspectionRecord({
    required this.productId,
    this.modelId,
    this.modelNumber,
    this.displayOrder,
    this.inspectionResult,
    this.inspectedBy,
    this.inspectedAt,
  });

  factory InspectorInspectionRecord.fromJson(Map<String, dynamic> json) {
    DateTime? parseDate(dynamic raw) {
      if (raw == null) return null;

      // 文字列の場合（time.RFC3339 など）
      if (raw is String && raw.isNotEmpty) {
        try {
          return DateTime.parse(raw);
        } catch (_) {
          return null;
        }
      }

      return null;
    }

    int? parseIntOrNull(dynamic raw) {
      if (raw == null) return null;
      if (raw is int) return raw;
      if (raw is num) return raw.toInt();
      if (raw is String) return int.tryParse(raw.trim());
      return null;
    }

    return InspectorInspectionRecord(
      productId: (json['productId'] ?? '') as String,
      modelId: json['modelId'] as String?,
      modelNumber: json['modelNumber'] as String?,
      displayOrder: parseIntOrNull(json['displayOrder']),
      inspectionResult: json['inspectionResult'] as String?,
      inspectedBy: json['inspectedBy'] as String?,
      inspectedAt: parseDate(json['inspectedAt']),
    );
  }
}
