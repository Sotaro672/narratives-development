// frontend/inspector/lib/models/inspector_product_detail.dart
import 'inspector_color.dart';
import 'inspector_product_blueprint.dart';
import 'inspector_inspection_record.dart';

class InspectorProductDetail {
  final String productId;
  final String productionId;
  final String modelId;
  final String productBlueprintId;

  final String modelNumber;
  final String size;
  final Map<String, int> measurements;
  final InspectorColor color;

  final InspectorProductBlueprint productBlueprint;
  final List<InspectorInspectionRecord> inspections;

  /// 現在の検品ステータス
  final String inspectionResult;

  InspectorProductDetail({
    required this.productId,
    required this.productionId,
    required this.modelId,
    required this.productBlueprintId,
    required this.modelNumber,
    required this.size,
    required this.measurements,
    required this.color,
    required this.productBlueprint,
    required this.inspections,
    required this.inspectionResult,
  });

  factory InspectorProductDetail.fromJson(Map<String, dynamic> json) {
    Map<String, int> parseMeasurements(dynamic raw) {
      if (raw is Map<String, dynamic>) {
        return raw.map(
          (key, value) => MapEntry(
            key,
            (value is num)
                ? value.toInt()
                : int.tryParse(value.toString()) ?? 0,
          ),
        );
      }
      return const {};
    }

    final inspectionsJson = (json['inspections'] as List<dynamic>?) ?? const [];
    final inspections = inspectionsJson
        .whereType<Map<String, dynamic>>()
        .map(InspectorInspectionRecord.fromJson)
        .toList();

    return InspectorProductDetail(
      productId: (json['productId'] ?? '') as String,
      productionId: (json['productionId'] ?? '') as String,
      modelId: (json['modelId'] ?? '') as String,
      productBlueprintId: (json['productBlueprintId'] ?? '') as String,
      modelNumber: (json['modelNumber'] ?? '') as String,
      size: (json['size'] ?? '') as String,
      measurements: parseMeasurements(json['measurements']),
      color: InspectorColor.fromJson(
        (json['color'] as Map<String, dynamic>? ?? const {}),
      ),
      productBlueprint: InspectorProductBlueprint.fromJson(
        (json['productBlueprint'] as Map<String, dynamic>? ?? const {}),
      ),
      inspections: inspections,
      inspectionResult: (json['inspectionResult'] ?? '') as String,
    );
  }
}
