// frontend/inspector/lib/models/inspector_inspection_batch.dart
import 'inspector_inspection_record.dart';

class InspectorInspectionBatch {
  final String productionId;
  final String status;
  final int quantity; // ★ inspection.batch.quantity
  final int totalPassed; // ★ inspection.batch.totalPassed
  final List<InspectorInspectionRecord> inspections;

  InspectorInspectionBatch({
    required this.productionId,
    required this.status,
    required this.quantity,
    required this.totalPassed,
    required this.inspections,
  });

  factory InspectorInspectionBatch.fromJson(Map<String, dynamic> json) {
    int parseInt(dynamic raw) {
      if (raw is int) return raw;
      if (raw is num) return raw.toInt();
      if (raw is String) return int.tryParse(raw) ?? 0;
      return 0;
    }

    final inspectionsJson = (json['inspections'] as List<dynamic>?) ?? const [];
    final inspections = inspectionsJson
        .whereType<Map<String, dynamic>>()
        .map(InspectorInspectionRecord.fromJson)
        .toList();

    return InspectorInspectionBatch(
      productionId: (json['productionId'] ?? '') as String,
      status: (json['status'] ?? '') as String,
      quantity: parseInt(json['quantity']),
      totalPassed: parseInt(json['totalPassed']),
      inspections: inspections,
    );
  }
}
