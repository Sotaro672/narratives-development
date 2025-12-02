// すでにある import に加えて、まだなら追加
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
      // "0xff112233" など文字列なら int に変換
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
  final String companyId;
  final String brandId;
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
    required this.companyId,
    required this.brandId,
    required this.itemType,
    required this.fit,
    required this.material,
    required this.weight,
    required this.qualityAssurance,
    required this.productIdTagType,
    required this.assigneeId,
  });

  factory InspectorProductBlueprint.fromJson(Map<String, dynamic> json) {
    return InspectorProductBlueprint(
      id: (json['id'] ?? '') as String,
      productName: (json['productName'] ?? '') as String,
      companyId: (json['companyId'] ?? '') as String,
      brandId: (json['brandId'] ?? '') as String,
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
    DateTime? parseDate(String? s) {
      if (s == null || s.isEmpty) return null;
      try {
        return DateTime.parse(s);
      } catch (_) {
        return null;
      }
    }

    return InspectorInspectionRecord(
      productId: (json['productId'] ?? '') as String,
      inspectionResult: json['inspectionResult'] as String?,
      inspectedBy: json['inspectedBy'] as String?,
      inspectedAt: parseDate(json['inspectedAt'] as String?),
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

  final InspectorProductBlueprint blueprint;
  final List<InspectorInspectionRecord> inspections;

  /// 現在の検品ステータス（`inspectionResult` の集約値など）
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
    required this.blueprint,
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
      blueprint: InspectorProductBlueprint.fromJson(
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

  /// 検品詳細画面用の情報を取得する API 呼び出し
  ///
  /// ※実際のエンドポイント名はバックエンド実装に合わせて変更してください。
  ///   ここでは例として `/inspector/products/{productId}` を想定しています。
  static Future<InspectorProductDetail> fetchInspectorDetail(
    String productId,
  ) async {
    final token = await _getIdToken();

    final uri = Uri.parse('$_baseUrl/inspector/products/$productId');
    final resp = await http.get(
      uri,
      headers: {
        'Authorization': 'Bearer $token',
        'Content-Type': 'application/json',
      },
    );

    if (resp.statusCode != 200) {
      throw Exception('検品詳細の取得に失敗しました: ${resp.statusCode} ${resp.body}');
    }

    final body = json.decode(resp.body) as Map<String, dynamic>;
    return InspectorProductDetail.fromJson(body);
  }

  /// 合否送信
  static Future<void> submitInspection({
    required String productId,
    required String result, // "passed" or "failed"
  }) async {
    final token = await _getIdToken();

    // バックエンドの仕様に合わせて `productionId` なども必要なら別途解決してください。
    // ここではシンプルに `/products/{id}` PATCH を叩く例です。
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

  /// 検品を完了する
  ///
  /// Go 側の想定ロジック:
  /// - 該当 productionId の inspections のうち inspectionResult == "notYet" を
  ///   "notManufactured" に更新
  /// - status を "completed"（または "inspected" など、ドメインに合わせた値）へ更新
  static Future<void> completeInspection({required String productionId}) async {
    final token = await _getIdToken();

    // エンドポイントは Go 側で実装予定のものに合わせてください。
    // 例として PATCH /products/inspections/complete を想定。
    final uri = Uri.parse('$_baseUrl/products/inspections/complete');

    final body = json.encode({
      'productionId': productionId,
      // status は Go 側で決め打ちでも良いので、ここでは送らず最低限の情報だけ渡す。
      // 必要なら 'status': 'completed' 等を追加してください。
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
      throw Exception('検品完了処理に失敗しました: ${resp.statusCode} ${resp.body}');
    }
  }
}
