// frontend\sns\lib\features\productBlueprint\infrastructure\product_bluleprint_repository_http.dart
import 'dart:convert';
import 'dart:io';

import 'package:http/http.dart' as http;

/// SNS productBlueprint response
/// backend: GET /sns/product-blueprints/{id}
class SnsProductBlueprintResponse {
  SnsProductBlueprintResponse({
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
    required this.printed,
  });

  final String id;

  final String productName;
  final String companyId;
  final String brandId;

  final String itemType;
  final String fit;
  final String material;
  final num? weight;

  final List<String> qualityAssurance;
  final String productIdTagType;

  final bool printed;

  factory SnsProductBlueprintResponse.fromJson(Map<String, dynamic> j) {
    final qaRaw = j['qualityAssurance'];
    final qa = (qaRaw is List)
        ? qaRaw.map((e) => e.toString()).toList()
        : <String>[];

    // productIdTag: { type: "qr" }
    String tagType = '';
    final tag = j['productIdTag'];
    if (tag is Map) {
      final t = tag['type'];
      if (t != null) tagType = t.toString();
    }

    return SnsProductBlueprintResponse(
      id: (j['id'] ?? '').toString(),
      productName: (j['productName'] ?? '').toString(),
      companyId: (j['companyId'] ?? '').toString(),
      brandId: (j['brandId'] ?? '').toString(),
      itemType: (j['itemType'] ?? '').toString(),
      fit: (j['fit'] ?? '').toString(),
      material: (j['material'] ?? '').toString(),
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

  // ✅ ここをあなたの SNS API base に合わせる
  // 既に他Repositoryで同様の定数があるなら、それを import して使うのが理想
  static const String _defaultBaseUrl =
      'https://narratives-backend-871263659099.asia-northeast1.run.app';

  Future<SnsProductBlueprintResponse> fetchProductBlueprintById(
    String productBlueprintId, {
    String baseUrl = _defaultBaseUrl,
  }) async {
    final id = productBlueprintId.trim();
    if (id.isEmpty) {
      throw ArgumentError('productBlueprintId is empty');
    }

    final uri = Uri.parse('$baseUrl/sns/product-blueprints/$id');

    final res = await _client.get(
      uri,
      headers: {HttpHeaders.acceptHeader: 'application/json'},
    );

    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw HttpException(
        'fetchProductBlueprintById failed: ${res.statusCode} ${res.body}',
        uri: uri,
      );
    }

    final data = jsonDecode(res.body);
    if (data is! Map<String, dynamic>) {
      throw const FormatException('invalid json shape (expected object)');
    }
    return SnsProductBlueprintResponse.fromJson(data);
  }
}
