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
  const SnsInventoryModelStock({
    required this.products,
    required this.accumulation,
  });

  final Map<String, bool> products;
  final int accumulation;

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

    int asInt(dynamic v, {int def = 0}) {
      if (v is int) return v;
      if (v is num) return v.toInt();
      if (v is String) return int.tryParse(v.trim()) ?? def;
      return def;
    }

    return SnsInventoryModelStock(
      products: products,
      accumulation: asInt(json['accumulation']),
    );
  }
}

@immutable
class SnsInventoryResponse {
  const SnsInventoryResponse({
    required this.id,
    required this.tokenBlueprintId,
    required this.productBlueprintId,
    required this.modelIds,
    required this.stock,
  });

  final String id;
  final String tokenBlueprintId;
  final String productBlueprintId;
  final List<String> modelIds;

  /// modelId -> stock
  final Map<String, SnsInventoryModelStock> stock;

  factory SnsInventoryResponse.fromJson(Map<String, dynamic> json) {
    final id = (json['id'] ?? '').toString().trim();
    final tb = (json['tokenBlueprintId'] ?? '').toString().trim();
    final pb = (json['productBlueprintId'] ?? '').toString().trim();

    final modelIdsRaw = json['modelIds'];
    final modelIds = <String>[];
    if (modelIdsRaw is List) {
      for (final v in modelIdsRaw) {
        final s = v.toString().trim();
        if (s.isNotEmpty) modelIds.add(s);
      }
    }

    final stockRaw = json['stock'];
    final Map<String, SnsInventoryModelStock> stock = {};
    if (stockRaw is Map) {
      for (final e in stockRaw.entries) {
        final modelId = e.key.toString().trim();
        if (modelId.isEmpty) continue;
        final v = e.value;
        if (v is Map) {
          stock[modelId] = SnsInventoryModelStock.fromJson(
            v.cast<String, dynamic>(),
          );
        }
      }
    }

    return SnsInventoryResponse(
      id: id,
      tokenBlueprintId: tb,
      productBlueprintId: pb,
      modelIds: modelIds,
      stock: stock,
    );
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
