// frontend/sns/lib/features/cart/infrastructure/cart_repository_http.dart
import 'dart:convert';

import 'package:http/http.dart' as http;

/// Buyer-facing Cart repository (HTTP).
///
/// Backend endpoints (CartHandler):
/// - GET    /sns/cart?avatarId=...
/// - POST   /sns/cart/items           body: {avatarId, modelId, qty}
/// - PUT    /sns/cart/items           body: {avatarId, modelId, qty}
/// - DELETE /sns/cart/items           body: {avatarId, modelId}
/// - DELETE /sns/cart?avatarId=...
/// - POST   /sns/cart/ordered         body: {avatarId}
///
/// NOTE:
/// - For now we send avatarId in query/body. (Header X-Avatar-Id is also supported by backend.)
/// - This repository uses a per-instance http.Client. Call dispose().
class CartRepositoryHttp {
  CartRepositoryHttp({http.Client? client, String? apiBase})
    : _client = client ?? http.Client(),
      _apiBase = (apiBase ?? const String.fromEnvironment('API_BASE')).trim();

  final http.Client _client;
  final String _apiBase;

  void dispose() {
    _client.close();
  }

  // ----------------------------
  // Public API
  // ----------------------------

  Future<CartDTO> fetchCart({required String avatarId}) async {
    final aid = avatarId.trim();
    if (aid.isEmpty) throw ArgumentError('avatarId is required');

    final uri = _uri('/sns/cart', qp: {'avatarId': aid});
    final res = await _client.get(uri, headers: _headersJson());

    return _decodeCart(res);
  }

  Future<CartDTO> addItem({
    required String avatarId,
    required String modelId,
    int qty = 1,
  }) async {
    final aid = avatarId.trim();
    final mid = modelId.trim();
    if (aid.isEmpty) throw ArgumentError('avatarId is required');
    if (mid.isEmpty) throw ArgumentError('modelId is required');
    if (qty <= 0) throw ArgumentError('qty must be >= 1');

    final uri = _uri('/sns/cart/items');
    final body = jsonEncode({'avatarId': aid, 'modelId': mid, 'qty': qty});

    final res = await _client.post(uri, headers: _headersJson(), body: body);
    return _decodeCart(res);
  }

  /// Sets quantity for a modelId.
  /// - qty <= 0 is treated as remove by backend.
  Future<CartDTO> setItemQty({
    required String avatarId,
    required String modelId,
    required int qty,
  }) async {
    final aid = avatarId.trim();
    final mid = modelId.trim();
    if (aid.isEmpty) throw ArgumentError('avatarId is required');
    if (mid.isEmpty) throw ArgumentError('modelId is required');

    final uri = _uri('/sns/cart/items');
    final body = jsonEncode({'avatarId': aid, 'modelId': mid, 'qty': qty});

    final res = await _client.put(uri, headers: _headersJson(), body: body);
    return _decodeCart(res);
  }

  Future<CartDTO> removeItem({
    required String avatarId,
    required String modelId,
  }) async {
    final aid = avatarId.trim();
    final mid = modelId.trim();
    if (aid.isEmpty) throw ArgumentError('avatarId is required');
    if (mid.isEmpty) throw ArgumentError('modelId is required');

    final uri = _uri('/sns/cart/items');
    final body = jsonEncode({'avatarId': aid, 'modelId': mid, 'qty': 0});

    // NOTE: http.delete supports body in recent http package versions.
    // If your version doesn't, switch to Request('DELETE', uri) and send manually.
    final res = await _client.delete(uri, headers: _headersJson(), body: body);
    return _decodeCart(res);
  }

  Future<void> clearCart({required String avatarId}) async {
    final aid = avatarId.trim();
    if (aid.isEmpty) throw ArgumentError('avatarId is required');

    final uri = _uri('/sns/cart', qp: {'avatarId': aid});
    final res = await _client.delete(uri, headers: _headersJson());

    if (res.statusCode == 204) return;
    _throwHttpError(res);
  }

  Future<CartDTO> markOrdered({required String avatarId}) async {
    final aid = avatarId.trim();
    if (aid.isEmpty) throw ArgumentError('avatarId is required');

    final uri = _uri('/sns/cart/ordered');
    final body = jsonEncode({'avatarId': aid});

    final res = await _client.post(uri, headers: _headersJson(), body: body);
    return _decodeCart(res);
  }

  // ----------------------------
  // DTO + parsing
  // ----------------------------

  CartDTO _decodeCart(http.Response res) {
    if (res.statusCode >= 200 && res.statusCode < 300) {
      final map = _decodeJsonMap(res.body);
      // backend returns {"error": "..."} for errors; ignore here (2xx only)
      return CartDTO.fromJson(map);
    }
    _throwHttpError(res);
    // unreachable
    throw StateError('unreachable');
  }

  Map<String, dynamic> _decodeJsonMap(String body) {
    final raw = body.trim().isEmpty ? '{}' : body;
    final v = jsonDecode(raw);
    if (v is Map<String, dynamic>) return v;
    throw FormatException('invalid json response');
  }

  void _throwHttpError(http.Response res) {
    final status = res.statusCode;
    String msg = 'HTTP $status';
    try {
      final m = _decodeJsonMap(res.body);
      final e = (m['error'] ?? '').toString().trim();
      if (e.isNotEmpty) msg = e;
    } catch (_) {
      final s = res.body.trim();
      if (s.isNotEmpty) msg = s;
    }
    throw CartHttpException(statusCode: status, message: msg);
  }

  // ----------------------------
  // URL / headers
  // ----------------------------

  Uri _uri(String path, {Map<String, String>? qp}) {
    final base = _apiBase;
    if (base.isEmpty) {
      throw StateError(
        'API_BASE is not set (use --dart-define=API_BASE=https://...)',
      );
    }

    final b = Uri.parse(base);
    final cleanPath = path.startsWith('/') ? path : '/$path';

    // join paths safely
    final joinedPath = _joinPaths(b.path, cleanPath);

    return Uri(
      scheme: b.scheme,
      userInfo: b.userInfo,
      host: b.host,
      port: b.hasPort ? b.port : null,
      path: joinedPath,
      queryParameters: qp?.isEmpty == true ? null : qp,
      fragment: b.fragment.isEmpty ? null : b.fragment,
    );
  }

  String _joinPaths(String a, String b) {
    final aa = a.trim();
    final bb = b.trim();
    if (aa.isEmpty || aa == '/') return bb;
    if (bb.isEmpty || bb == '/') return aa;
    if (aa.endsWith('/') && bb.startsWith('/')) return aa + bb.substring(1);
    if (!aa.endsWith('/') && !bb.startsWith('/')) return '$aa/$bb';
    return aa + bb;
  }

  Map<String, String> _headersJson() => const {
    'Content-Type': 'application/json; charset=utf-8',
    'Accept': 'application/json',
  };
}

// ----------------------------
// Models
// ----------------------------

class CartDTO {
  CartDTO({
    required this.avatarId,
    required this.items,
    required this.createdAt,
    required this.updatedAt,
    required this.expiresAt,
    required this.ordered,
  });

  final String avatarId;

  /// modelId -> qty
  final Map<String, int> items;

  final DateTime? createdAt;
  final DateTime? updatedAt;
  final DateTime? expiresAt;

  final bool ordered;

  int totalQty() {
    var sum = 0;
    for (final v in items.values) {
      if (v > 0) sum += v;
    }
    return sum;
  }

  factory CartDTO.fromJson(Map<String, dynamic> json) {
    final aid = (json['avatarId'] ?? '').toString().trim();

    final itemsRaw = json['items'];
    final Map<String, int> items = {};
    if (itemsRaw is Map) {
      for (final entry in itemsRaw.entries) {
        final k = entry.key.toString().trim();
        final v = entry.value;
        final n = (v is int) ? v : int.tryParse(v.toString());
        if (k.isNotEmpty && n != null && n > 0) {
          items[k] = n;
        }
      }
    }

    return CartDTO(
      avatarId: aid,
      items: items,
      createdAt: _tryParseTime(json['createdAt']),
      updatedAt: _tryParseTime(json['updatedAt']),
      expiresAt: _tryParseTime(json['expiresAt']),
      ordered: (json['ordered'] == true),
    );
  }

  static DateTime? _tryParseTime(dynamic v) {
    final s = (v ?? '').toString().trim();
    if (s.isEmpty) return null;
    return DateTime.tryParse(s);
  }
}

class CartHttpException implements Exception {
  CartHttpException({required this.statusCode, required this.message});

  final int statusCode;
  final String message;

  @override
  String toString() =>
      'CartHttpException(statusCode=$statusCode, message=$message)';
}
