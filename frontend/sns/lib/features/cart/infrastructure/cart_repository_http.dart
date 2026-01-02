// frontend/sns/lib/features/cart/infrastructure/cart_repository_http.dart
import 'dart:convert';

import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http;

import '../../inventory/infrastructure/inventory_repository_http.dart';

/// Buyer-facing Cart repository (HTTP).
///
/// Backend endpoints (CartHandler):
/// - GET    /sns/cart?avatarId=...
/// - POST   /sns/cart/items           body: {avatarId, inventoryId, listId, modelId, qty}
/// - PUT    /sns/cart/items           body: {avatarId, inventoryId, listId, modelId, qty}
/// - DELETE /sns/cart/items           body: {avatarId, inventoryId, listId, modelId}
/// - DELETE /sns/cart?avatarId=...
///
/// NOTE:
/// - Web(CORS) ではカスタムヘッダ（例: x-avatar-id）を送ると preflight が走り、
///   backend 側で Access-Control-Allow-Headers に許可が無いとブロックされます。
/// - そのため avatarId は **query/body のみ**で送ります（ヘッダには入れない）。
/// - /sns/cart/items は backend 実装差異を踏まえ、query には avatarId を付けず **body のみに統一**します。
class CartRepositoryHttp {
  CartRepositoryHttp({http.Client? client, String? apiBase})
    : _client = client ?? http.Client(),
      _apiBase = (apiBase ?? const String.fromEnvironment('API_BASE')).trim();

  final http.Client _client;

  /// Optional override. If empty, resolveSnsApiBase() will be used.
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
    final res = await _sendAuthed('GET', uri);

    return _decodeCart(res);
  }

  Future<CartDTO> addItem({
    required String avatarId,
    required String inventoryId,
    required String listId,
    required String modelId,
    int qty = 1,
  }) async {
    final aid = avatarId.trim();
    final invId = inventoryId.trim();
    final lid = listId.trim();
    final mid = modelId.trim();

    if (aid.isEmpty) throw ArgumentError('avatarId is required');
    if (invId.isEmpty) throw ArgumentError('inventoryId is required');
    if (lid.isEmpty) throw ArgumentError('listId is required');
    if (mid.isEmpty) throw ArgumentError('modelId is required');
    if (qty <= 0) throw ArgumentError('qty must be >= 1');

    // ✅ /sns/cart/items は query に avatarId を付けず、body のみに統一
    final uri = _uri('/sns/cart/items');
    final bodyMap = <String, dynamic>{
      'avatarId': aid,
      'inventoryId': invId,
      'listId': lid,
      'modelId': mid,
      'qty': qty,
    };
    final body = jsonEncode(bodyMap);

    // ignore: avoid_print
    print('[CartRepositoryHttp] POST $uri body=$body');

    final res = await _sendAuthed('POST', uri, body: body);
    return _decodeCart(res);
  }

  /// Sets quantity for a cart item (identified by inventoryId/listId/modelId).
  /// - qty <= 0 is treated as remove by backend.
  Future<CartDTO> setItemQty({
    required String avatarId,
    required String inventoryId,
    required String listId,
    required String modelId,
    required int qty,
  }) async {
    final aid = avatarId.trim();
    final invId = inventoryId.trim();
    final lid = listId.trim();
    final mid = modelId.trim();

    if (aid.isEmpty) throw ArgumentError('avatarId is required');
    if (invId.isEmpty) throw ArgumentError('inventoryId is required');
    if (lid.isEmpty) throw ArgumentError('listId is required');
    if (mid.isEmpty) throw ArgumentError('modelId is required');

    final uri = _uri('/sns/cart/items');
    final bodyMap = <String, dynamic>{
      'avatarId': aid,
      'inventoryId': invId,
      'listId': lid,
      'modelId': mid,
      'qty': qty,
    };
    final body = jsonEncode(bodyMap);

    // ignore: avoid_print
    print('[CartRepositoryHttp] PUT $uri body=$body');

    final res = await _sendAuthed('PUT', uri, body: body);
    return _decodeCart(res);
  }

  Future<CartDTO> removeItem({
    required String avatarId,
    required String inventoryId,
    required String listId,
    required String modelId,
  }) async {
    final aid = avatarId.trim();
    final invId = inventoryId.trim();
    final lid = listId.trim();
    final mid = modelId.trim();

    if (aid.isEmpty) throw ArgumentError('avatarId is required');
    if (invId.isEmpty) throw ArgumentError('inventoryId is required');
    if (lid.isEmpty) throw ArgumentError('listId is required');
    if (mid.isEmpty) throw ArgumentError('modelId is required');

    final uri = _uri('/sns/cart/items');
    final bodyMap = <String, dynamic>{
      'avatarId': aid,
      'inventoryId': invId,
      'listId': lid,
      'modelId': mid,
    };
    final body = jsonEncode(bodyMap);

    // ignore: avoid_print
    print('[CartRepositoryHttp] DELETE $uri body=$body');

    final res = await _sendAuthed('DELETE', uri, body: body);
    return _decodeCart(res);
  }

  Future<void> clearCart({required String avatarId}) async {
    final aid = avatarId.trim();
    if (aid.isEmpty) throw ArgumentError('avatarId is required');

    final uri = _uri('/sns/cart', qp: {'avatarId': aid});

    // ignore: avoid_print
    print('[CartRepositoryHttp] DELETE $uri (clear)');

    final res = await _sendAuthed('DELETE', uri);

    if (res.statusCode == 204) return;
    _throwHttpError(res);
  }

  // ----------------------------
  // Auth / sending
  // ----------------------------

  Future<Map<String, String>> _headersJsonAuthed({
    bool forceRefreshToken = false,
  }) async {
    final u = FirebaseAuth.instance.currentUser;
    if (u == null) {
      throw CartHttpException(statusCode: 401, message: 'not_signed_in');
    }

    final idToken = await u.getIdToken(forceRefreshToken);
    final tok = (idToken ?? '').trim(); // ✅ null-safe

    if (tok.isEmpty) {
      throw CartHttpException(statusCode: 401, message: 'empty_id_token');
    }

    // ✅ Authorization は “simple header” ではないので Web では preflight が走る
    // ただし /sns/payment が通っているので、backend 側は Allow-Headers に Authorization が入っている前提。
    return <String, String>{'Authorization': 'Bearer $tok', ..._headersJson()};
  }

  /// Sends request with Firebase Authorization header.
  /// - If 401, retry once with forceRefreshToken=true.
  Future<http.Response> _sendAuthed(
    String method,
    Uri uri, {
    String? body,
  }) async {
    http.Response res;

    final h1 = await _headersJsonAuthed(forceRefreshToken: false);
    res = await _sendRaw(method, uri, headers: h1, body: body);

    if (res.statusCode != 401) return res;

    // ignore: avoid_print
    print(
      '[CartRepositoryHttp] 401 received -> retry with refreshed token. uri=$uri',
    );

    final h2 = await _headersJsonAuthed(forceRefreshToken: true);
    return _sendRaw(method, uri, headers: h2, body: body);
  }

  Future<http.Response> _sendRaw(
    String method,
    Uri uri, {
    required Map<String, String> headers,
    String? body,
  }) async {
    final m = method.trim().toUpperCase();

    // body なし
    if (body == null) {
      switch (m) {
        case 'GET':
          return _client.get(uri, headers: headers);
        case 'DELETE':
          return _client.delete(uri, headers: headers);
        case 'POST':
          return _client.post(uri, headers: headers);
        case 'PUT':
          return _client.put(uri, headers: headers);
        default:
          final req = http.Request(m, uri);
          req.headers.addAll(headers);
          final streamed = await _client.send(req);
          return http.Response.fromStream(streamed);
      }
    }

    // body あり
    switch (m) {
      case 'POST':
        return _client.post(uri, headers: headers, body: body);
      case 'PUT':
        return _client.put(uri, headers: headers, body: body);
      case 'DELETE':
        // ✅ http.delete(body) が環境/バージョンで不安定なため Request で送る
        final req = http.Request('DELETE', uri);
        req.headers.addAll(headers);
        req.body = body;
        final streamed = await _client.send(req);
        return http.Response.fromStream(streamed);
      default:
        final req = http.Request(m, uri);
        req.headers.addAll(headers);
        req.body = body;
        final streamed = await _client.send(req);
        return http.Response.fromStream(streamed);
    }
  }

  // ----------------------------
  // DTO + parsing
  // ----------------------------

  CartDTO _decodeCart(http.Response res) {
    if (res.statusCode >= 200 && res.statusCode < 300) {
      final map = _decodeJsonMap(res.body);
      return CartDTO.fromJson(map);
    }
    _throwHttpError(res);
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

    // ignore: avoid_print
    print('[CartRepositoryHttp] HTTP error status=$status body="${res.body}"');

    throw CartHttpException(statusCode: status, message: msg);
  }

  // ----------------------------
  // URL / headers
  // ----------------------------

  Uri _uri(String path, {Map<String, String>? qp}) {
    final baseRaw = (_apiBase.isNotEmpty ? _apiBase : resolveSnsApiBase())
        .trim();

    if (baseRaw.isEmpty) {
      throw StateError(
        'API_BASE is not set (use --dart-define=API_BASE=https://...)',
      );
    }

    final base = baseRaw.replaceAll(RegExp(r'\/+$'), '');

    final b = Uri.parse(base);
    final cleanPath = path.startsWith('/') ? path : '/$path';
    final joinedPath = _joinPaths(b.path, cleanPath);

    return Uri(
      scheme: b.scheme,
      userInfo: b.userInfo,
      host: b.host,
      port: b.hasPort ? b.port : null,
      path: joinedPath,
      queryParameters: (qp == null || qp.isEmpty) ? null : qp,
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

  /// ✅ CORS 的に “simple headers” 寄りにする（x- 系などカスタムは入れない）
  /// NOTE: Authorization はここには入れない（authed 時は _headersJsonAuthed で追加）
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
  });

  final String avatarId;

  /// itemKey -> item
  final Map<String, CartItemDTO> items;

  final DateTime? createdAt;
  final DateTime? updatedAt;
  final DateTime? expiresAt;

  int totalQty() {
    var sum = 0;
    for (final it in items.values) {
      final q = it.qty;
      if (q > 0) sum += q;
    }
    return sum;
  }

  factory CartDTO.fromJson(Map<String, dynamic> json) {
    final aid = (json['avatarId'] ?? json['id'] ?? '').toString().trim();

    final itemsRaw = json['items'];
    final Map<String, CartItemDTO> items = {};
    if (itemsRaw is Map) {
      for (final entry in itemsRaw.entries) {
        final key = entry.key.toString().trim();
        final v = entry.value;

        if (key.isEmpty) continue;

        // New shape: itemKey -> {inventoryId, listId, modelId, qty}
        if (v is Map) {
          final it = CartItemDTO.fromJson(v.cast<String, dynamic>());
          if (it.isValid) items[key] = it;
          continue;
        }

        // Legacy shape: modelId -> qty
        final n = (v is int) ? v : int.tryParse(v.toString());
        if (n != null && n > 0) {
          items[key] = CartItemDTO(
            inventoryId: '',
            listId: '',
            modelId: key,
            qty: n,
          );
        }
      }
    }

    return CartDTO(
      avatarId: aid,
      items: items,
      createdAt: _tryParseTime(json['createdAt']),
      updatedAt: _tryParseTime(json['updatedAt']),
      expiresAt: _tryParseTime(json['expiresAt']),
    );
  }

  static DateTime? _tryParseTime(dynamic v) {
    if (v == null) return null;

    if (v is String) {
      final s = v.trim();
      if (s.isEmpty) return null;
      return DateTime.tryParse(s);
    }

    if (v is Map) {
      final sec = v['seconds'];
      final nanos = v['nanos'] ?? 0;

      final s = (sec is int) ? sec : int.tryParse(sec?.toString() ?? '');
      final n = (nanos is int)
          ? nanos
          : (int.tryParse(nanos?.toString() ?? '0') ?? 0);

      if (s == null) return null;

      return DateTime.fromMillisecondsSinceEpoch(
        s * 1000 + (n ~/ 1000000),
        isUtc: true,
      );
    }

    final s = v.toString().trim();
    if (s.isEmpty) return null;
    return DateTime.tryParse(s);
  }
}

class CartItemDTO {
  CartItemDTO({
    required this.inventoryId,
    required this.listId,
    required this.modelId,
    required this.qty,
  });

  final String inventoryId;
  final String listId;
  final String modelId;
  final int qty;

  bool get isValid =>
      inventoryId.trim().isNotEmpty &&
      listId.trim().isNotEmpty &&
      modelId.trim().isNotEmpty &&
      qty > 0;

  static int _toInt(dynamic v) {
    if (v == null) return 0;
    if (v is int) return v;
    if (v is double) return v.toInt();
    if (v is num) return v.toInt();
    final s = v.toString().trim();
    if (s.isEmpty) return 0;
    return int.tryParse(s) ?? 0;
  }

  factory CartItemDTO.fromJson(Map<String, dynamic> json) {
    final invId = (json['inventoryId'] ?? '').toString().trim();
    final lid = (json['listId'] ?? '').toString().trim();
    final mid = (json['modelId'] ?? '').toString().trim();
    final qty = _toInt(json['qty']);

    return CartItemDTO(inventoryId: invId, listId: lid, modelId: mid, qty: qty);
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
