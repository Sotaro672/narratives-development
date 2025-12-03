import 'dart:convert';
import 'package:http/http.dart' as http;
import 'package:firebase_auth/firebase_auth.dart';

// ================================================
// 検品アプリ用 DTO
// ================================================

class InspectorColor {
  final int rgb; // ARGB or RGB int
  final String? name;

  InspectorColor({required this.rgb, this.name});

  factory InspectorColor.fromJson(Map<String, dynamic> json) {
    final raw = json['rgb'];
    int rgbValue;

    if (raw is int) {
      rgbValue = raw;
    } else if (raw is String) {
      rgbValue = int.tryParse(raw) ?? 0xff000000;
    } else {
      rgbValue = 0xff000000;
    }

    return InspectorColor(rgb: rgbValue, name: json['name'] as String?);
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

class InspectorInspectionRecord {
  final String productId;
  final String? inspectionResult;
  final String? inspectedBy;
  final DateTime? inspectedAt;

  InspectorInspectionRecord({
    required this.productId,
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

      // Firestore Timestamp ライクなオブジェクトの場合もここで拡張可能
      // 例: { "seconds": 123, "nanos": 0 }

      return null;
    }

    return InspectorInspectionRecord(
      productId: (json['productId'] ?? '') as String,
      inspectionResult: json['inspectionResult'] as String?,
      inspectedBy: json['inspectedBy'] as String?,
      inspectedAt: parseDate(json['inspectedAt']),
    );
  }
}

/// inspections テーブル 1 レコード分（= 1 productionId 分）のバッチ
class InspectorInspectionBatch {
  final String productionId;
  final String status;
  final List<InspectorInspectionRecord> inspections;

  InspectorInspectionBatch({
    required this.productionId,
    required this.status,
    required this.inspections,
  });

  factory InspectorInspectionBatch.fromJson(Map<String, dynamic> json) {
    final inspectionsJson = (json['inspections'] as List<dynamic>?) ?? const [];
    final inspections = inspectionsJson
        .whereType<Map<String, dynamic>>()
        .map(InspectorInspectionRecord.fromJson)
        .toList();

    return InspectorInspectionBatch(
      productionId: (json['productionId'] ?? '') as String,
      status: (json['status'] ?? '') as String,
      inspections: inspections,
    );
  }
}

/// 検品詳細画面用にまとめたデータ
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

    // ※ /inspector/products/{id} 側で inspections を返さない場合はここは空になる
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

// ================================================
// ProductApi
// ================================================

class ProductApi {
  static const String _baseUrl =
      'https://narratives-backend-871263659099.asia-northeast1.run.app';

  static Future<String> _getIdToken() async {
    final user = FirebaseAuth.instance.currentUser;
    final token = await user?.getIdToken();
    if (token == null || token.isEmpty) {
      throw Exception('ログイン情報が見つかりません（IDトークンが取得できませんでした）');
    }
    return token;
  }

  /// 検品詳細取得 API（productId から詳細取得）
  /// 1. GET /inspector/products/{productId} で product / model / blueprint 情報を取得
  /// 2. 取得した productionId を使って GET /products/inspections?productionId=xxx を叩き、
  ///    inspections テーブルの一覧をマージする
  static Future<InspectorProductDetail> fetchInspectorDetail(
    String productId,
  ) async {
    final token = await _getIdToken();

    // 1) プロダクト詳細（+ productionId 等）を取得
    final detailUri = Uri.parse('$_baseUrl/inspector/products/$productId');

    final detailResp = await http.get(
      detailUri,
      headers: {'Authorization': 'Bearer $token'},
    );

    if (detailResp.statusCode != 200) {
      throw Exception(
        '検品詳細の取得に失敗しました: ${detailResp.statusCode} ${detailResp.body}',
      );
    }

    final detailBody = json.decode(detailResp.body) as Map<String, dynamic>;
    final baseDetail = InspectorProductDetail.fromJson(detailBody);

    // 2) productionId から inspections バッチを取得
    InspectorInspectionBatch? batch;
    try {
      batch = await fetchInspectionBatch(baseDetail.productionId);
    } catch (e) {
      // inspections 取得に失敗しても、詳細自体は返す
      // （必要ならここで log 出力など）
      batch = null;
    }

    if (batch == null) {
      return baseDetail;
    }

    // 3) この productId 自身の最新検査結果を拾って inspectionResult として反映
    final recordsForThisProduct = batch.inspections
        .where((r) => r.productId == baseDetail.productId)
        .toList();

    final currentResult = recordsForThisProduct.isNotEmpty
        ? (recordsForThisProduct.first.inspectionResult ?? '')
        : baseDetail.inspectionResult;

    // 4) baseDetail をベースにして、inspections と inspectionResult を上書きしたものを返す
    return InspectorProductDetail(
      productId: baseDetail.productId,
      productionId: baseDetail.productionId,
      modelId: baseDetail.modelId,
      productBlueprintId: baseDetail.productBlueprintId,
      modelNumber: baseDetail.modelNumber,
      size: baseDetail.size,
      measurements: baseDetail.measurements,
      color: baseDetail.color,
      productBlueprint: baseDetail.productBlueprint,
      // 検品履歴としては、同じ productionId に属する productId 一覧を全部出したいので
      // バッチ側の inspections をそのまま渡す
      inspections: batch.inspections,
      inspectionResult: currentResult,
    );
  }

  /// 同じ productionId を持つ inspections テーブルのバッチを取得
  ///
  /// GET /products/inspections?productionId={productionId}
  static Future<InspectorInspectionBatch> fetchInspectionBatch(
    String productionId,
  ) async {
    final token = await _getIdToken();

    final uri = Uri.parse(
      '$_baseUrl/products/inspections?productionId=$productionId',
    );

    final resp = await http.get(
      uri,
      headers: {'Authorization': 'Bearer $token'},
    );

    if (resp.statusCode != 200) {
      throw Exception('inspections 取得に失敗しました: ${resp.statusCode} ${resp.body}');
    }

    final body = json.decode(resp.body) as Map<String, dynamic>;
    return InspectorInspectionBatch.fromJson(body);
  }

  /// products テーブルの検品結果（単体）を更新
  static Future<void> submitInspection({
    required String productId,
    required String result,
  }) async {
    final token = await _getIdToken();

    final uri = Uri.parse('$_baseUrl/products/$productId');
    final now = DateTime.now().toUtc().toIso8601String();

    final user = FirebaseAuth.instance.currentUser;
    final inspectedBy = user?.email ?? user?.uid ?? 'unknown';

    final body = json.encode({
      'inspectionResult': result == 'passed' ? 'passed' : 'failed',
      'inspectedAt': now,
      'inspectedBy': inspectedBy,
    });

    final resp = await http.patch(
      uri,
      headers: {
        'Authorization': 'Bearer $token',
        'Content-Type': 'application/json',
      },
      body: body,
    );

    if (resp.statusCode != 200) {
      throw Exception('検品結果の送信に失敗しました: ${resp.statusCode} ${resp.body}');
    }
  }

  /// inspections テーブルの検品結果を更新する API
  static Future<void> updateInspectionBatch({
    required String productionId,
    required String productId,
    required String inspectionResult,
  }) async {
    final token = await _getIdToken();

    final uri = Uri.parse('$_baseUrl/products/inspections');
    final now = DateTime.now().toUtc().toIso8601String();

    final user = FirebaseAuth.instance.currentUser;
    final inspectedBy = user?.email ?? user?.uid ?? 'unknown';

    final body = json.encode({
      'productionId': productionId,
      'productId': productId,
      'inspectionResult': inspectionResult,
      'inspectedBy': inspectedBy,
      'inspectedAt': now,
    });

    final resp = await http.patch(
      uri,
      headers: {
        'Authorization': 'Bearer $token',
        'Content-Type': 'application/json',
      },
      body: body,
    );

    if (resp.statusCode != 200) {
      throw Exception('inspections 更新に失敗しました: ${resp.statusCode} ${resp.body}');
    }
  }

  /// 検品完了
  static Future<void> completeInspection({required String productionId}) async {
    final token = await _getIdToken();

    final uri = Uri.parse('$_baseUrl/products/inspections/complete');

    final body = json.encode({'productionId': productionId});

    final resp = await http.patch(
      uri,
      headers: {
        'Authorization': 'Bearer $token',
        'Content-Type': 'application/json',
      },
      body: body,
    );

    if (resp.statusCode != 200) {
      throw Exception('検品完了処理に失敗しました: ${resp.statusCode} ${resp.body}');
    }
  }
}
