// frontend/mall/lib/features/productBlueprint/infrastructure/product_blueprint_repository_http.dart
import 'dart:convert';

import 'package:http/http.dart' as http;

// ✅ API_BASE 解決ロジック（single source of truth）
import '../../../app/config/api_base.dart';

/// Mall productBlueprint response
/// backend: GET /mall/product-blueprints/{id}
class MallProductBlueprintResponse {
  MallProductBlueprintResponse({
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

  factory MallProductBlueprintResponse.fromJson(Map<String, dynamic> j) {
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
      if (tagType.isEmpty) tagType = s(tag['Type']);
    } else if (tag is Map) {
      // Map<dynamic,dynamic> 等も拾う
      tagType = s(tag['type']);
      if (tagType.isEmpty) tagType = s(tag['Type']);
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

    return MallProductBlueprintResponse(
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

  static String _normalizeBaseUrl(String s) {
    s = s.trim();
    if (s.isEmpty) return s;
    while (s.endsWith('/')) {
      s = s.substring(0, s.length - 1);
    }
    return s;
  }

  Map<String, String> _jsonHeaders() => const {'Accept': 'application/json'};

  Future<MallProductBlueprintResponse> fetchProductBlueprintById(
    String productBlueprintId, {
    String? baseUrl,
  }) async {
    final id = productBlueprintId.trim();
    if (id.isEmpty) {
      throw ArgumentError('productBlueprintId is empty');
    }

    final b = _normalizeBaseUrl(
      (baseUrl ?? '').trim().isNotEmpty ? baseUrl!.trim() : resolveSnsApiBase(),
    );

    final uri = Uri.parse('$b/mall/product-blueprints/$id');

    final res = await _client.get(uri, headers: _jsonHeaders());

    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw HttpException(
        'fetchProductBlueprintById failed: ${res.statusCode}',
        url: uri.toString(),
        body: res.body,
      );
    }

    final decoded = jsonDecode(res.body);

    // wrapper 吸収: {data:{...}} を許容
    if (decoded is Map) {
      final m = decoded.cast<String, dynamic>();
      final data = m['data'];
      if (data is Map<String, dynamic>) {
        return MallProductBlueprintResponse.fromJson(data);
      }
      if (data is Map) {
        return MallProductBlueprintResponse.fromJson(
          Map<String, dynamic>.from(data),
        );
      }
      return MallProductBlueprintResponse.fromJson(m);
    }

    throw const FormatException('invalid json shape (expected object)');
  }
}

class HttpException implements Exception {
  HttpException(this.message, {this.url, this.body});

  final String message;
  final String? url;
  final String? body;

  @override
  String toString() {
    final u = url == null ? '' : ' url=$url';
    final b = body == null
        ? ''
        : ' body=${body!.length > 300 ? body!.substring(0, 300) : body}';
    return 'HttpException($message$u$b)';
  }
}
