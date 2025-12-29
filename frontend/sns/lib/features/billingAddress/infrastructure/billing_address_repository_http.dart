// frontend/sns/lib/features/billingAddress/infrastructure/billing_address_repository_http.dart
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;
import 'package:firebase_auth/firebase_auth.dart';

/// ✅ API_BASE に統一（shippingAddress と同じ）
///
/// Priority:
/// 1) override (constructor)
/// 2) --dart-define=API_BASE=https://...
/// 3) --dart-define=API_BASE_URL=https://... (互換)
/// 4) fallback
const String _fallbackBaseUrl =
    'https://narratives-backend-871263659099.asia-northeast1.run.app';

String _resolveApiBase({String? override}) {
  final o = (override ?? '').trim();
  if (o.isNotEmpty) return _normalizeBase(o);

  const v1 = String.fromEnvironment('API_BASE', defaultValue: '');
  final s1 = v1.trim();
  if (s1.isNotEmpty) return _normalizeBase(s1);

  const v2 = String.fromEnvironment('API_BASE_URL', defaultValue: '');
  final s2 = v2.trim();
  if (s2.isNotEmpty) return _normalizeBase(s2);

  return _normalizeBase(_fallbackBaseUrl);
}

String _normalizeBase(String base) {
  final b = base.trim();
  if (b.endsWith('/')) return b.substring(0, b.length - 1);
  return b;
}

/// Simple HTTP repository for SNS billing address endpoints.
///
/// Endpoints (sns):
/// - POST   /sns/billing-addresses
/// - PATCH  /sns/billing-addresses/{id}
/// - DELETE /sns/billing-addresses/{id}
/// - GET    /sns/billing-addresses/{id}
class BillingAddressRepositoryHttp {
  BillingAddressRepositoryHttp({
    http.Client? client,
    FirebaseAuth? auth,
    String? baseUrl,
  }) : _client = client ?? http.Client(),
       _auth = auth ?? FirebaseAuth.instance,
       _base = _resolveApiBase(override: baseUrl) {
    if (_base.trim().isEmpty) {
      throw Exception(
        'API_BASE is not set (use --dart-define=API_BASE=https://...)',
      );
    }
    _log('[BillingAddressRepositoryHttp] init baseUrl=$_base');
  }

  final http.Client _client;
  final FirebaseAuth _auth;

  final String _base;

  // ✅ release でもログを出したい場合: --dart-define=ENABLE_HTTP_LOG=true
  static const bool _envHttpLog = bool.fromEnvironment(
    'ENABLE_HTTP_LOG',
    defaultValue: false,
  );

  bool get _logEnabled => kDebugMode || _envHttpLog;

  Uri _uri(String path, [Map<String, String>? query]) {
    final p = path.startsWith('/') ? path : '/$path';
    return Uri.parse('$_base$p').replace(queryParameters: query);
  }

  Future<Map<String, String>> _authHeaders() async {
    final headers = <String, String>{'Accept': 'application/json'};

    try {
      final u = _auth.currentUser;
      if (u != null) {
        final token = await u.getIdToken();
        headers['Authorization'] = 'Bearer $token';
      }
    } catch (e) {
      _log('[BillingAddressRepositoryHttp] token error: $e');
    }

    return headers;
  }

  // ---------------------------------------------------------------------------
  // Public API
  // ---------------------------------------------------------------------------

  /// GET /sns/billing-addresses/{id}
  Future<BillingAddressDTO> getById({required String id}) async {
    final rid = id.trim();
    if (rid.isEmpty) {
      throw ArgumentError('id is empty');
    }

    final uri = _uri('/sns/billing-addresses/$rid');

    final headers = await _authHeaders();
    _logRequest('GET', uri, headers: headers, payload: null);

    final res = await _client.get(uri, headers: headers);

    _logResponse('GET', uri, res.statusCode, res.body);

    final body = res.body;
    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw HttpException(
        statusCode: res.statusCode,
        message: _extractError(body) ?? 'request failed',
        url: uri.toString(),
      );
    }

    final decoded = _decodeObject(body);
    return BillingAddressDTO.fromJson(decoded);
  }

  /// POST /sns/billing-addresses
  ///
  /// NOTE:
  /// - userId は原則 server 側で uid から決める想定（送らない）
  Future<BillingAddressDTO> create({
    required String cardNumber,
    required String cardholderName,
    required String cvc,
  }) async {
    final payload = CreateBillingAddressRequest(
      cardNumber: cardNumber,
      cardholderName: cardholderName,
      cvc: cvc,
    );

    final uri = _uri('/sns/billing-addresses');

    final headers = await _authHeaders();
    headers['Content-Type'] = 'application/json';

    _logRequest(
      'POST',
      uri,
      headers: headers,
      payload: _maskSensitivePayload(payload.toJson()),
    );

    final res = await _client.post(
      uri,
      headers: headers,
      body: jsonEncode(payload.toJson()),
    );

    _logResponse('POST', uri, res.statusCode, res.body);

    final body = res.body;
    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw HttpException(
        statusCode: res.statusCode,
        message: _extractError(body) ?? 'request failed',
        url: uri.toString(),
      );
    }

    final decoded = _decodeObject(body);
    return BillingAddressDTO.fromJson(decoded);
  }

  /// PATCH /sns/billing-addresses/{id}
  ///
  /// ✅ shippingAddress と同様に、backend が upsert 挙動 (200 or 201) でもOK。
  /// ただし body が空のケースに備えて、空なら GET で取り直す。
  Future<BillingAddressDTO> update({
    required String id,
    String? cardNumber,
    String? cardholderName,
    String? cvc,
  }) async {
    final rid = id.trim();
    if (rid.isEmpty) {
      throw ArgumentError('id is empty');
    }

    final payload = UpdateBillingAddressRequest(
      cardNumber: cardNumber,
      cardholderName: cardholderName,
      cvc: cvc,
    );

    final uri = _uri('/sns/billing-addresses/$rid');

    final headers = await _authHeaders();
    headers['Content-Type'] = 'application/json';

    final bodyJson = payload.toJson();
    _logRequest(
      'PATCH',
      uri,
      headers: headers,
      payload: _maskSensitivePayload(bodyJson),
    );

    final res = await _client.patch(
      uri,
      headers: headers,
      body: jsonEncode(bodyJson),
    );

    _logResponse('PATCH', uri, res.statusCode, res.body);

    final body = res.body;

    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw HttpException(
        statusCode: res.statusCode,
        message: _extractError(body) ?? 'request failed',
        url: uri.toString(),
      );
    }

    // ✅ 念のため: body が空なら GET で取り直す（将来の 204/空返却対策）
    if (body.trim().isEmpty) {
      return getById(id: rid);
    }

    final decoded = _decodeObject(body);
    return BillingAddressDTO.fromJson(decoded);
  }

  /// DELETE /sns/billing-addresses/{id}
  Future<void> delete({required String id}) async {
    final rid = id.trim();
    if (rid.isEmpty) {
      throw ArgumentError('id is empty');
    }

    final uri = _uri('/sns/billing-addresses/$rid');

    final headers = await _authHeaders();
    _logRequest('DELETE', uri, headers: headers, payload: null);

    final res = await _client.delete(uri, headers: headers);

    _logResponse('DELETE', uri, res.statusCode, res.body);

    final body = res.body;
    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw HttpException(
        statusCode: res.statusCode,
        message: _extractError(body) ?? 'request failed',
        url: uri.toString(),
      );
    }
  }

  void dispose() {
    _client.close();
  }

  // ---------------------------------------------------------------------------
  // helpers
  // ---------------------------------------------------------------------------

  Map<String, dynamic> _decodeObject(String body) {
    final b = body.trim();
    if (b.isEmpty) {
      throw const FormatException('Empty response body (expected object)');
    }
    final decoded = jsonDecode(b);
    if (decoded is Map<String, dynamic>) return decoded;
    if (decoded is Map) return Map<String, dynamic>.from(decoded);
    throw const FormatException('Invalid JSON shape (expected object)');
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

  // ---------------------------------------------------------------------------
  // Logging (debug or ENABLE_HTTP_LOG=true)
  // ---------------------------------------------------------------------------

  void _log(String msg) {
    if (!_logEnabled) return;
    debugPrint(msg);
  }

  void _logRequest(
    String method,
    Uri uri, {
    required Map<String, String> headers,
    required Map<String, dynamic>? payload,
  }) {
    if (!_logEnabled) return;

    // Authorization は伏せる
    final safeHeaders = <String, String>{};
    headers.forEach((k, v) {
      if (k.toLowerCase() == 'authorization') {
        safeHeaders[k] = 'Bearer ***';
      } else {
        safeHeaders[k] = v;
      }
    });

    final b = StringBuffer();
    b.writeln('[BillingAddressRepositoryHttp] request');
    b.writeln('  method=$method');
    b.writeln('  url=$uri');
    b.writeln('  headers=${jsonEncode(safeHeaders)}');
    if (payload != null) {
      b.writeln('  payload=${_truncate(jsonEncode(payload), 1500)}');
    }
    debugPrint(b.toString());
  }

  void _logResponse(String method, Uri uri, int status, String body) {
    if (!_logEnabled) return;

    final truncated = _truncate(body, 1500);
    final b = StringBuffer();
    b.writeln('[BillingAddressRepositoryHttp] response');
    b.writeln('  method=$method');
    b.writeln('  url=$uri');
    b.writeln('  status=$status');
    if (truncated.isNotEmpty) {
      // ✅ 念のため response も cardNumber/cvc を伏せたいが、
      // backend が返さない設計に寄せるのが本筋。
      b.writeln('  body=$truncated');
    }
    debugPrint(b.toString());
  }

  String _truncate(String s, int max) {
    final t = s.trim();
    if (t.length <= max) {
      return t;
    }
    return '${t.substring(0, max)}...(truncated ${t.length - max} chars)';
  }

  Map<String, dynamic> _maskSensitivePayload(Map<String, dynamic> src) {
    final m = Map<String, dynamic>.from(src);

    if (m.containsKey('cardNumber')) {
      m['cardNumber'] = _maskCardNumber(m['cardNumber']?.toString());
    }
    if (m.containsKey('cvc')) {
      final v = (m['cvc'] ?? '').toString().trim();
      if (v.isNotEmpty) {
        m['cvc'] = '***';
      }
    }

    return m;
  }

  String _maskCardNumber(String? v) {
    final s = (v ?? '').replaceAll(RegExp(r'[^0-9]'), '');
    if (s.isEmpty) {
      return '';
    }
    final last4 = s.length >= 4 ? s.substring(s.length - 4) : s;
    return '**** **** **** $last4';
  }
}

// -----------------------------------------------------------------------------
// DTOs / Requests
// -----------------------------------------------------------------------------
/// Mirrors backend billingAddress entity (client-safe).
///
/// ✅ IMPORTANT:
/// - client は生カード番号 / 生CVC を扱わない前提。
/// - backend が `cardNumberMasked` / `cvcMasked` を返さない場合は空文字にします
///   （生値 `cardNumber` / `cvc` へのフォールバックはしない）。
@immutable
class BillingAddressDTO {
  const BillingAddressDTO({
    required this.id,
    required this.userId,
    required this.cardNumberMasked,
    required this.cardholderName,
    required this.cvcMasked,
    required this.createdAt,
    required this.updatedAt,
  });

  final String id;
  final String userId;

  /// Backend should NOT return raw card number.
  final String cardNumberMasked;

  final String cardholderName;

  /// Backend should NOT return raw CVC.
  final String cvcMasked;

  final DateTime createdAt;
  final DateTime updatedAt;

  factory BillingAddressDTO.fromJson(Map<String, dynamic> json) {
    String s(dynamic v) => (v ?? '').toString().trim();

    // ✅ ここは “生値へフォールバックしない”
    final maskedNum = s(json['cardNumberMasked']);
    final maskedCvc = s(json['cvcMasked']);

    return BillingAddressDTO(
      id: s(json['id']),
      userId: s(json['userId']),
      cardNumberMasked: maskedNum,
      cardholderName: s(json['cardholderName']),
      cvcMasked: maskedCvc,
      createdAt: _parseDateTime(json['createdAt']),
      updatedAt: _parseDateTime(json['updatedAt']),
    );
  }

  static DateTime _parseDateTime(dynamic v) {
    final s = (v ?? '').toString().trim();
    if (s.isEmpty) {
      return DateTime.fromMillisecondsSinceEpoch(0, isUtc: true);
    }
    return DateTime.parse(s).toUtc();
  }
}

/// POST body
@immutable
class CreateBillingAddressRequest {
  CreateBillingAddressRequest({
    required String cardNumber,
    required String cardholderName,
    required String cvc,
  }) : cardNumber = cardNumber.trim(),
       cardholderName = cardholderName.trim(),
       cvc = cvc.trim();

  final String cardNumber;
  final String cardholderName;
  final String cvc;

  Map<String, dynamic> toJson() => {
    'cardNumber': cardNumber,
    'cardholderName': cardholderName,
    'cvc': cvc,
  };
}

/// PATCH body (partial update)
@immutable
class UpdateBillingAddressRequest {
  UpdateBillingAddressRequest({
    String? cardNumber,
    String? cardholderName,
    String? cvc,
  }) : cardNumber = cardNumber?.trim(),
       cardholderName = cardholderName?.trim(),
       cvc = cvc?.trim();

  final String? cardNumber;
  final String? cardholderName;
  final String? cvc;

  Map<String, dynamic> toJson() {
    final m = <String, dynamic>{};

    // NOTE: 空文字は送らない（消去仕様が必要なら別途決める）
    if (cardNumber != null && cardNumber!.isNotEmpty) {
      m['cardNumber'] = cardNumber;
    }
    if (cardholderName != null && cardholderName!.isNotEmpty) {
      m['cardholderName'] = cardholderName;
    }
    if (cvc != null && cvc!.isNotEmpty) {
      m['cvc'] = cvc;
    }

    return m;
  }
}

@immutable
class HttpException implements Exception {
  const HttpException({
    required this.statusCode,
    required this.message,
    required this.url,
  });

  final int statusCode;
  final String message;
  final String url;

  @override
  String toString() => 'HttpException($statusCode) $message ($url)';
}
