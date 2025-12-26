// frontend/sns/lib/features/inventory/infrastructure/inventory_repository_http.dart
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;

/// SNS buyer-facing API base URL.
///
/// Priority:
/// 1) --dart-define=API_BASE_URL=https://...
/// 2) fallback
const String _fallbackBaseUrl =
    'https://narratives-backend-871263659099.asia-northeast1.run.app';

String _resolveApiBase() {
  const fromDefine = String.fromEnvironment('API_BASE_URL');
  final base = (fromDefine.isNotEmpty ? fromDefine : _fallbackBaseUrl).trim();
  return base.endsWith('/') ? base.substring(0, base.length - 1) : base;
}

@immutable
class SnsInventoryModelStock {
  const SnsInventoryModelStock({required this.products});

  /// productId -> true
  /// ※ backend が products を返さない（stockKeys only）場合は空になる
  final Map<String, bool> products;

  factory SnsInventoryModelStock.fromJson(Map<String, dynamic> json) {
    final rawProducts = json['products'];
    final Map<String, bool> products = {};
    if (rawProducts is Map) {
      for (final e in rawProducts.entries) {
        final k = e.key.toString().trim();
        final v = e.value;
        if (k.isEmpty) continue;
        products[k] = v == true;
      }
    }
    return SnsInventoryModelStock(products: products);
  }
}

@immutable
class SnsInventoryResponse {
  const SnsInventoryResponse({
    required this.id,
    required this.tokenBlueprintId,
    required this.productBlueprintId,
    required this.modelIds,
    required this.stockKeys,
    required this.stock,
  });

  final String id;
  final String tokenBlueprintId;
  final String productBlueprintId;

  /// UI が “モデル一覧” 表示に使う想定
  /// - backend が modelIds を返さない場合は stockKeys から補完する
  final List<String> modelIds;

  /// ✅ stockKeys（modelId の集合）
  /// - backend が stockKeys-only を返す場合もここが埋まる
  final List<String> stockKeys;

  /// modelId -> stock detail（products）
  /// - backend が stockKeys-only の場合は、キーだけ拾って空 products を入れる
  final Map<String, SnsInventoryModelStock> stock;

  factory SnsInventoryResponse.fromJson(Map<String, dynamic> json) {
    final id = (json['id'] ?? '').toString().trim();
    final tb = (json['tokenBlueprintId'] ?? '').toString().trim();
    final pb = (json['productBlueprintId'] ?? '').toString().trim();

    // -------------------------
    // stock / stockKeys
    // -------------------------
    final Map<String, SnsInventoryModelStock> stock = {};
    final List<String> stockKeys = [];

    final stockRaw = json['stock'];
    if (stockRaw is Map) {
      for (final e in stockRaw.entries) {
        final modelId = e.key.toString().trim();
        if (modelId.isEmpty) continue;

        stockKeys.add(modelId);

        final v = e.value;
        if (v is Map) {
          stock[modelId] = SnsInventoryModelStock.fromJson(
            v.cast<String, dynamic>(),
          );
        } else {
          // stockKeys-only（value が bool/int など）でもキーだけ拾って空の products を入れる
          stock[modelId] = const SnsInventoryModelStock(products: {});
        }
      }
    } else if (stockRaw is List) {
      // 念のため: stock が配列で返るケース
      for (final v in stockRaw) {
        final modelId = v.toString().trim();
        if (modelId.isEmpty) continue;
        stockKeys.add(modelId);
        stock[modelId] = const SnsInventoryModelStock(products: {});
      }
    }

    final normalizedStockKeys = _uniqPreserveOrder(stockKeys);

    // -------------------------
    // modelIds（なければ stockKeys から補完）
    // -------------------------
    final modelIdsRaw = json['modelIds'];
    final modelIds = <String>[];
    if (modelIdsRaw is List) {
      for (final v in modelIdsRaw) {
        final s = v.toString().trim();
        if (s.isNotEmpty) modelIds.add(s);
      }
    }
    final normalizedModelIds = modelIds.isNotEmpty
        ? _uniqPreserveOrder(modelIds)
        : normalizedStockKeys;

    return SnsInventoryResponse(
      id: id,
      tokenBlueprintId: tb,
      productBlueprintId: pb,
      modelIds: normalizedModelIds,
      stockKeys: normalizedStockKeys,
      stock: stock,
    );
  }

  static List<String> _uniqPreserveOrder(List<String> xs) {
    final seen = <String>{};
    final out = <String>[];
    for (final x in xs) {
      final s = x.trim();
      if (s.isEmpty) continue;
      if (seen.add(s)) out.add(s);
    }
    return out;
  }
}

class InventoryRepositoryHttp {
  InventoryRepositoryHttp({http.Client? client})
    : _client = client ?? http.Client();

  final http.Client _client;

  String get _base => _resolveApiBase();

  Uri _uri(String path, [Map<String, String>? query]) {
    final p = path.startsWith('/') ? path : '/$path';
    return Uri.parse('$_base$p').replace(queryParameters: query);
  }

  /// GET /sns/inventories/{id}
  Future<SnsInventoryResponse> fetchInventoryById(String id) async {
    final invId = id.trim();
    if (invId.isEmpty) {
      throw ArgumentError('id is required');
    }

    final uri = _uri('/sns/inventories/$invId');
    final res = await _client.get(
      uri,
      headers: const {'Accept': 'application/json'},
    );

    final body = res.body;
    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw SnsHttpException(
        statusCode: res.statusCode,
        message: _extractError(body) ?? 'request failed',
        url: uri.toString(),
      );
    }

    final decoded = jsonDecode(body);
    if (decoded is! Map<String, dynamic>) {
      throw FormatException('Invalid JSON shape (expected object)');
    }
    return SnsInventoryResponse.fromJson(decoded);
  }

  /// GET /sns/inventories?productBlueprintId=...&tokenBlueprintId=...
  Future<SnsInventoryResponse> fetchInventoryByQuery({
    required String productBlueprintId,
    required String tokenBlueprintId,
  }) async {
    final pb = productBlueprintId.trim();
    final tb = tokenBlueprintId.trim();
    if (pb.isEmpty || tb.isEmpty) {
      throw ArgumentError(
        'productBlueprintId and tokenBlueprintId are required',
      );
    }

    final uri = _uri('/sns/inventories', {
      'productBlueprintId': pb,
      'tokenBlueprintId': tb,
    });

    final res = await _client.get(
      uri,
      headers: const {'Accept': 'application/json'},
    );

    final body = res.body;
    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw SnsHttpException(
        statusCode: res.statusCode,
        message: _extractError(body) ?? 'request failed',
        url: uri.toString(),
      );
    }

    final decoded = jsonDecode(body);
    if (decoded is! Map<String, dynamic>) {
      throw FormatException('Invalid JSON shape (expected object)');
    }
    return SnsInventoryResponse.fromJson(decoded);
  }

  void dispose() {
    _client.close();
  }

  String? _extractError(String body) {
    try {
      final decoded = jsonDecode(body);
      if (decoded is Map && decoded['error'] != null) {
        return decoded['error'].toString();
      }
    } catch (_) {
      // ignore
    }
    return null;
  }
}

@immutable
class SnsHttpException implements Exception {
  const SnsHttpException({
    required this.statusCode,
    required this.message,
    required this.url,
  });

  final int statusCode;
  final String message;
  final String url;

  @override
  String toString() => 'SnsHttpException($statusCode) $message ($url)';
}
