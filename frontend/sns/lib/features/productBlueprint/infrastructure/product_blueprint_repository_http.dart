// frontend\sns\lib\features\productBlueprint\infrastructure\product_bluleprint_repository_http.dart
import 'dart:convert';

import 'package:http/http.dart' as http;

/// SNS productBlueprint response
/// backend: GET /sns/product-blueprints/{id}
class SnsProductBlueprintResponse {
  SnsProductBlueprintResponse({
    required this.id,
    required this.productName,
    required this.companyId,
    required this.companyName,
    required this.brandId,
    required this.brandName,
    required this.itemType,
    required this.fit,
    required this.material,
    required this.weight,
    required this.qualityAssurance,
    required this.productIdTagType,
    required this.printed,
  });

  final String id;

  final String productName;

  final String companyId;
  final String companyName; // resolved

  final String brandId;
  final String brandName; // resolved

  final String itemType;
  final String fit;
  final String material;
  final num? weight;

  final List<String> qualityAssurance;

  /// productIdTag.type を取り出したもの（無ければ空文字）
  final String productIdTagType;

  final bool printed;

  factory SnsProductBlueprintResponse.fromJson(Map<String, dynamic> j) {
    String s(dynamic v) => (v ?? '').toString().trim();

    final qaRaw = j['qualityAssurance'];
    final qa = (qaRaw is List)
        ? qaRaw.map((e) => e.toString()).toList()
        : <String>[];

    // --- productIdTag: best-effort ---
    // 期待: { "productIdTag": { "type": "qr" } }
    // 互換: { "productIdTagType": "qr" } / { "productIdTag": "qr" }
    String tagType = '';

    // 1) 正: productIdTag.type
    final tag = j['productIdTag'];
    if (tag is Map<String, dynamic>) {
      tagType = s(tag['type']);
    } else if (tag is Map) {
      // Map<dynamic,dynamic> 等も拾う
      tagType = s(tag['type']);
      if (tagType.isEmpty && tag.containsKey('Type')) {
        tagType = s(tag['Type']);
      }
    } else if (tag != null) {
      // 万一 backend が productIdTag を string で返していた場合
      tagType = s(tag);
    }

    // 2) フラット字段 fallback
    if (tagType.isEmpty) {
      tagType = s(j['productIdTagType']);
    }
    if (tagType.isEmpty) {
      tagType = s(j['productIdTag_type']); // 念のため
    }

    return SnsProductBlueprintResponse(
      id: s(j['id']),
      productName: s(j['productName']),
      companyId: s(j['companyId']),
      companyName: s(j['companyName']),
      brandId: s(j['brandId']),
      brandName: s(j['brandName']),
      itemType: s(j['itemType']),
      fit: s(j['fit']),
      material: s(j['material']),
      weight: (j['weight'] is num) ? (j['weight'] as num) : null,
      qualityAssurance: qa,
      productIdTagType: tagType,
      printed: j['printed'] == true,
    );
  }
}

class ProductBlueprintRepositoryHttp {
  ProductBlueprintRepositoryHttp({http.Client? client})
    : _client = client ?? http.Client();

  final http.Client _client;

  void dispose() {
    _client.close();
  }

  static String _resolveApiBase() {
    const env = String.fromEnvironment("API_BASE");
    final s = env.trim();
    if (s.isNotEmpty) return s;

    return "https://narratives-backend-871263659099.asia-northeast1.run.app";
  }

  static String _normalizeBaseUrl(String s) {
    s = s.trim();
    if (s.isEmpty) return s;
    while (s.endsWith("/")) {
      s = s.substring(0, s.length - 1);
    }
    return s;
  }

  Map<String, String> _jsonHeaders() => const {"Accept": "application/json"};

  Future<SnsProductBlueprintResponse> fetchProductBlueprintById(
    String productBlueprintId, {
    String? baseUrl,
  }) async {
    final id = productBlueprintId.trim();
    if (id.isEmpty) {
      throw ArgumentError('productBlueprintId is empty');
    }

    final b = _normalizeBaseUrl(
      (baseUrl ?? '').trim().isNotEmpty ? baseUrl! : _resolveApiBase(),
    );

    final uri = Uri.parse('$b/sns/product-blueprints/$id');

    final res = await _client.get(uri, headers: _jsonHeaders());

    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw HttpException(
        'fetchProductBlueprintById failed: ${res.statusCode}',
        url: uri.toString(),
        body: res.body,
      );
    }

    final data = jsonDecode(res.body);
    if (data is! Map<String, dynamic>) {
      throw const FormatException('invalid json shape (expected object)');
    }
    return SnsProductBlueprintResponse.fromJson(data);
  }
}

class HttpException implements Exception {
  HttpException(this.message, {this.url, this.body});

  final String message;
  final String? url;
  final String? body;

  @override
  String toString() {
    final u = url == null ? "" : " url=$url";
    final b = body == null
        ? ""
        : " body=${body!.length > 300 ? body!.substring(0, 300) : body}";
    return "HttpException($message$u$b)";
  }
}
