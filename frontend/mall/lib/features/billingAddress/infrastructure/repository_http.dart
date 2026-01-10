//frontend\mall\lib\features\billingAddress\infrastructure\repository_http.dart
import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;
import 'package:firebase_auth/firebase_auth.dart';

import 'api.dart';

/// Simple HTTP repository for buyer billing address endpoints.
///
/// Endpoints (buyer):
/// - POST   /mall/billing-addresses
/// - PATCH  /mall/billing-addresses/{id}
/// - DELETE /mall/billing-addresses/{id}
/// - GET    /mall/billing-addresses/{id}
class BillingAddressRepositoryHttp {
  BillingAddressRepositoryHttp({
    http.Client? client,
    FirebaseAuth? auth,
    String? apiBase,
  }) : _api = ApiClient(
         tag: 'BillingAddressRepositoryHttp',
         client: client,
         auth: auth,
         apiBase: apiBase,
       );

  final ApiClient _api;

  /// GET /mall/billing-addresses/{id}
  Future<BillingAddressDTO> getById({required String id}) async {
    final rid = id.trim();
    if (rid.isEmpty) throw ArgumentError('id is empty');

    final uri = _api.uri('billing-addresses/$rid');
    final res = await _api.sendAuthed('GET', uri);

    _api.ensureSuccess(res, uri);

    final decoded = _api.decodeObject(res.body);
    final data = _api.unwrapData(decoded);
    return BillingAddressDTO.fromJson(data);
  }

  /// POST /mall/billing-addresses
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

    final uri = _api.uri('billing-addresses');

    final body = payload.toJson();
    final res = await _api.sendAuthed(
      'POST',
      uri,
      jsonBody: body,
      logPayload: _maskSensitivePayload(body),
    );

    _api.ensureSuccess(res, uri);

    final decoded = _api.decodeObject(res.body);
    final data = _api.unwrapData(decoded);
    return BillingAddressDTO.fromJson(data);
  }

  /// PATCH /mall/billing-addresses/{id}
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

    final uri = _api.uri('billing-addresses/$rid');
    final body = payload.toJson();

    final res = await _api.sendAuthed(
      'PATCH',
      uri,
      jsonBody: body,
      logPayload: _maskSensitivePayload(body),
    );

    _api.ensureSuccess(res, uri);

    if (res.body.trim().isEmpty) {
      return getById(id: rid);
    }

    final decoded = _api.decodeObject(res.body);
    final data = _api.unwrapData(decoded);
    return BillingAddressDTO.fromJson(data);
  }

  /// DELETE /mall/billing-addresses/{id}
  Future<void> delete({required String id}) async {
    final rid = id.trim();
    if (rid.isEmpty) throw ArgumentError('id is empty');

    final uri = _api.uri('billing-addresses/$rid');
    final res = await _api.sendAuthed('DELETE', uri);

    _api.ensureSuccess(res, uri);
  }

  void dispose() {
    _api.dispose();
  }

  // ---------------------------------------------------------------------------
  // Local helpers (domain-specific)
  // ---------------------------------------------------------------------------

  Map<String, dynamic> _maskSensitivePayload(Map<String, dynamic>? src) {
    if (src == null) return const <String, dynamic>{};
    final m = Map<String, dynamic>.from(src);

    if (m.containsKey('cardNumber')) {
      m['cardNumber'] = _maskCardNumber(m['cardNumber']?.toString());
    }
    if (m.containsKey('cvc')) {
      final v = (m['cvc'] ?? '').toString().trim();
      if (v.isNotEmpty) m['cvc'] = '***';
    }

    return m;
  }

  String _maskCardNumber(String? v) {
    final s = (v ?? '').replaceAll(RegExp(r'[^0-9]'), '');
    if (s.isEmpty) return '';
    final last4 = s.length >= 4 ? s.substring(s.length - 4) : s;
    return '**** **** **** $last4';
  }
}

// -----------------------------------------------------------------------------
// DTOs / Requests
// -----------------------------------------------------------------------------

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
  final String cardNumberMasked;
  final String cardholderName;
  final String cvcMasked;
  final DateTime createdAt;
  final DateTime updatedAt;

  factory BillingAddressDTO.fromJson(Map<String, dynamic> json) {
    String s(dynamic v) => (v ?? '').toString().trim();

    return BillingAddressDTO(
      id: s(json['id']),
      userId: s(json['userId']),
      cardNumberMasked: s(json['cardNumberMasked']),
      cardholderName: s(json['cardholderName']),
      cvcMasked: s(json['cvcMasked']),
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
