// frontend/sns/lib/features/billingAddress/infrastructure/billing_address_repository_http.dart
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;

/// SNS buyer-facing API base URL.
///
/// Priority:
/// 1) --dart-define=API_BASE_URL=https://...
/// 2) (fallback) Cloud Run default (edit as needed)
const String _fallbackBaseUrl =
    'https://narratives-backend-871263659099.asia-northeast1.run.app';

String _resolveApiBase() {
  const fromDefine = String.fromEnvironment('API_BASE_URL');
  final base = (fromDefine.isNotEmpty ? fromDefine : _fallbackBaseUrl).trim();
  if (base.endsWith('/')) {
    return base.substring(0, base.length - 1);
  }
  return base;
}

/// Simple HTTP repository for SNS billing address endpoints.
///
/// Endpoints (sns):
/// - POST   /sns/billing-addresses
/// - PATCH  /sns/billing-addresses/{id}
/// - DELETE /sns/billing-addresses/{id}
/// - GET    /sns/billing-addresses/{id}
class BillingAddressRepositoryHttp {
  BillingAddressRepositoryHttp({http.Client? client})
    : _client = client ?? http.Client();

  final http.Client _client;

  String get _base => _resolveApiBase();

  Uri _uri(String path, [Map<String, String>? query]) {
    final p = path.startsWith('/') ? path : '/$path';
    return Uri.parse('$_base$p').replace(queryParameters: query);
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

    _logRequest('GET', uri, null);

    final res = await _client.get(
      uri,
      headers: const {'Accept': 'application/json'},
    );

    _logResponse('GET', uri, res.statusCode, res.body);

    final body = res.body;
    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw HttpException(
        statusCode: res.statusCode,
        message: _extractError(body) ?? 'request failed',
        url: uri.toString(),
      );
    }

    final decoded = jsonDecode(body);
    if (decoded is! Map<String, dynamic>) {
      throw FormatException('Invalid JSON shape (expected object)');
    }
    return BillingAddressDTO.fromJson(decoded);
  }

  /// POST /sns/billing-addresses
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

    _logRequest('POST', uri, _maskSensitivePayload(payload.toJson()));

    final res = await _client.post(
      uri,
      headers: const {
        'Accept': 'application/json',
        'Content-Type': 'application/json',
      },
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

    final decoded = jsonDecode(body);
    if (decoded is! Map<String, dynamic>) {
      throw FormatException('Invalid JSON shape (expected object)');
    }
    return BillingAddressDTO.fromJson(decoded);
  }

  /// PATCH /sns/billing-addresses/{id}
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

    _logRequest('PATCH', uri, _maskSensitivePayload(payload.toJson()));

    final res = await _client.patch(
      uri,
      headers: const {
        'Accept': 'application/json',
        'Content-Type': 'application/json',
      },
      body: jsonEncode(payload.toJson()),
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

    final decoded = jsonDecode(body);
    if (decoded is! Map<String, dynamic>) {
      throw FormatException('Invalid JSON shape (expected object)');
    }
    return BillingAddressDTO.fromJson(decoded);
  }

  /// DELETE /sns/billing-addresses/{id}
  Future<void> delete({required String id}) async {
    final rid = id.trim();
    if (rid.isEmpty) {
      throw ArgumentError('id is empty');
    }

    final uri = _uri('/sns/billing-addresses/$rid');

    _logRequest('DELETE', uri, null);

    final res = await _client.delete(
      uri,
      headers: const {'Accept': 'application/json'},
    );

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
  // Logging (debug only)
  // ---------------------------------------------------------------------------

  void _logRequest(String method, Uri uri, Map<String, dynamic>? payload) {
    if (!kDebugMode) {
      return;
    }
    final b = StringBuffer();
    b.writeln('[BillingAddressRepositoryHttp] request');
    b.writeln('  method=$method');
    b.writeln('  url=$uri');
    if (payload != null) {
      b.writeln('  payload=${jsonEncode(payload)}');
    }
    debugPrint(b.toString());
  }

  void _logResponse(String method, Uri uri, int status, String body) {
    if (!kDebugMode) {
      return;
    }
    final truncated = _truncate(body, 1200);
    final b = StringBuffer();
    b.writeln('[BillingAddressRepositoryHttp] response');
    b.writeln('  method=$method');
    b.writeln('  url=$uri');
    b.writeln('  status=$status');
    if (truncated.isNotEmpty) {
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

/// Mirrors backend billingAddress entity (simplified).
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

    final maskedNum = s(json['cardNumberMasked']);
    final maskedCvc = s(json['cvcMasked']);

    return BillingAddressDTO(
      id: s(json['id']),
      userId: s(json['userId']),
      cardNumberMasked: maskedNum.isNotEmpty
          ? maskedNum
          : s(json['cardNumber']),
      cardholderName: s(json['cardholderName']),
      cvcMasked: maskedCvc.isNotEmpty ? maskedCvc : s(json['cvc']),
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
