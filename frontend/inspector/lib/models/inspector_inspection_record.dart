// frontend/inspector/lib/models/inspector_inspection_record.dart
class InspectorInspectionRecord {
  final String productId;
  // ★ InspectionItem の modelId
  final String? modelId;
  // ★ InspectionUsecase から渡される modelNumber を受け取る
  final String? modelNumber;
  final String? inspectionResult;
  final String? inspectedBy;
  final DateTime? inspectedAt;

  InspectorInspectionRecord({
    required this.productId,
    this.modelId,
    this.modelNumber,
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

    return InspectorInspectionRecord(
      productId: (json['productId'] ?? '') as String,
      modelId: json['modelId'] as String?,
      modelNumber: json['modelNumber'] as String?,
      inspectionResult: json['inspectionResult'] as String?,
      inspectedBy: json['inspectedBy'] as String?,
      inspectedAt: parseDate(json['inspectedAt']),
    );
  }
}
