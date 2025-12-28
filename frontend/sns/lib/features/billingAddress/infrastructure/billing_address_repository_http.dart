// frontend/sns/lib/features/billingAddress/infrastructure/billing_address_repository_http.dart
import 'dart:convert';

import 'package:dio/dio.dart';

/// BillingAddress API (sns backend)
///
/// Endpoints (assumed):
/// - POST   /billing-addresses
/// - PATCH  /billing-addresses/{id}
/// - DELETE /billing-addresses/{id}
/// - GET    /billing-addresses/{id}
///
/// Notes:
/// - This client uses Dio (recommended).
/// - API base is read from --dart-define=API_BASE=...
///   Example:
///     flutter run -d chrome --dart-define=API_BASE=https://your-api
class BillingAddressRepositoryHttp {
  BillingAddressRepositoryHttp({Dio? dio})
    : _dio =
          dio ??
          Dio(
            BaseOptions(
              baseUrl: _resolveApiBase(),
              connectTimeout: const Duration(seconds: 10),
              sendTimeout: const Duration(seconds: 20),
              receiveTimeout: const Duration(seconds: 20),
              headers: const {
                'Content-Type': 'application/json',
                'Accept': 'application/json',
              },
            ),
          );

  final Dio _dio;

  void dispose() {
    _dio.close(force: true);
  }

  // ---------------------------------------------------------------------------
  // Public API
  // ---------------------------------------------------------------------------

  /// GET /billing-addresses/{id}
  Future<BillingAddressDTO> getById({required String id}) async {
    final rid = id.trim();
    if (rid.isEmpty) throw ArgumentError('id is empty');

    final res = await _dio.get('/billing-addresses/$rid');
    return BillingAddressDTO.fromJson(_asMap(res.data));
  }

  /// POST /billing-addresses
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

    final res = await _dio.post('/billing-addresses', data: payload.toJson());

    return BillingAddressDTO.fromJson(_asMap(res.data));
  }

  /// PATCH /billing-addresses/{id}
  Future<BillingAddressDTO> update({
    required String id,
    String? cardNumber,
    String? cardholderName,
    String? cvc,
  }) async {
    final rid = id.trim();
    if (rid.isEmpty) throw ArgumentError('id is empty');

    final payload = UpdateBillingAddressRequest(
      cardNumber: cardNumber,
      cardholderName: cardholderName,
      cvc: cvc,
    );

    final res = await _dio.patch(
      '/billing-addresses/$rid',
      data: payload.toJson(),
    );

    return BillingAddressDTO.fromJson(_asMap(res.data));
  }

  /// DELETE /billing-addresses/{id}
  Future<void> delete({required String id}) async {
    final rid = id.trim();
    if (rid.isEmpty) throw ArgumentError('id is empty');

    await _dio.delete('/billing-addresses/$rid');
  }

  // ---------------------------------------------------------------------------
  // Helpers
  // ---------------------------------------------------------------------------

  static String _resolveApiBase() {
    const v = String.fromEnvironment('API_BASE', defaultValue: '');
    final s = v.trim();
    if (s.isEmpty) {
      throw Exception(
        'API_BASE is not set (use --dart-define=API_BASE=https://...)',
      );
    }
    // Ensure no trailing slash to keep paths consistent
    return s.endsWith('/') ? s.substring(0, s.length - 1) : s;
  }

  static Map<String, dynamic> _asMap(dynamic data) {
    if (data is Map<String, dynamic>) return data;
    if (data is String) {
      final decoded = jsonDecode(data);
      if (decoded is Map<String, dynamic>) return decoded;
    }
    throw StateError('Unexpected response body type: ${data.runtimeType}');
  }
}

// -----------------------------------------------------------------------------
// DTOs / Requests
// -----------------------------------------------------------------------------

/// Mirrors backend billingAddress entity (new simplified version).
class BillingAddressDTO {
  BillingAddressDTO({
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
  /// Expect something like "************1234" or "**** **** **** 1234".
  final String cardNumberMasked;

  final String cardholderName;

  /// Backend should NOT return raw CVC.
  /// Expect something like "***".
  final String cvcMasked;

  final DateTime createdAt;
  final DateTime updatedAt;

  factory BillingAddressDTO.fromJson(Map<String, dynamic> json) {
    return BillingAddressDTO(
      id: (json['id'] ?? '').toString(),
      userId: (json['userId'] ?? '').toString(),
      cardNumberMasked: (json['cardNumberMasked'] ?? json['cardNumber'] ?? '')
          .toString(),
      cardholderName: (json['cardholderName'] ?? '').toString(),
      cvcMasked: (json['cvcMasked'] ?? json['cvc'] ?? '').toString(),
      createdAt: _parseDateTime(json['createdAt']),
      updatedAt: _parseDateTime(json['updatedAt']),
    );
  }

  static DateTime _parseDateTime(dynamic v) {
    final s = (v ?? '').toString().trim();
    if (s.isEmpty) return DateTime.fromMillisecondsSinceEpoch(0, isUtc: true);
    return DateTime.parse(s).toUtc();
  }
}

/// POST body
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
    if (cardNumber != null) m['cardNumber'] = cardNumber;
    if (cardholderName != null) m['cardholderName'] = cardholderName;
    if (cvc != null) m['cvc'] = cvc;
    return m;
  }
}
