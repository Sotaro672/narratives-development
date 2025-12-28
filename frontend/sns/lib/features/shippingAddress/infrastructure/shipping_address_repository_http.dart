// frontend/sns/lib/features/shippingAddress/infrastructure/shipping_address_repository_http.dart
import 'package:dio/dio.dart';
import 'package:firebase_auth/firebase_auth.dart';

String _resolveApiBase() {
  const v = String.fromEnvironment('API_BASE', defaultValue: '');
  final s = v.trim();
  if (s.isEmpty) {
    throw Exception(
      'API_BASE is not set (use --dart-define=API_BASE=https://...)',
    );
  }
  return s;
}

/// Domain-ish model for SNS shipping address (matches backend shippingAddress.entity.go)
class ShippingAddress {
  ShippingAddress({
    required this.id,
    required this.userId,
    required this.zipCode,
    required this.state,
    required this.city,
    required this.street,
    required this.street2,
    required this.country,
    required this.createdAt,
    required this.updatedAt,
  });

  final String id;
  final String userId;
  final String zipCode;
  final String state;
  final String city;
  final String street;
  final String
  street2; // optional in UI, but backend entity currently has string (can be "")
  final String country;
  final DateTime createdAt;
  final DateTime updatedAt;

  factory ShippingAddress.fromJson(Map<String, dynamic> json) {
    DateTime parseTime(dynamic v) {
      if (v is String) {
        final s = v.trim();
        if (s.isNotEmpty) return DateTime.parse(s).toUtc();
      }
      // Firestore Timestamp etc are not expected on HTTP boundary; fallback
      return DateTime.fromMillisecondsSinceEpoch(0, isUtc: true);
    }

    String s(dynamic v) => (v ?? '').toString().trim();

    return ShippingAddress(
      id: s(json['id']),
      userId: s(json['userId']),
      zipCode: s(json['zipCode']),
      state: s(json['state']),
      city: s(json['city']),
      street: s(json['street']),
      street2: s(json['street2']),
      country: s(json['country']),
      createdAt: parseTime(json['createdAt']),
      updatedAt: parseTime(json['updatedAt']),
    );
  }

  Map<String, dynamic> toJson() => {
    'id': id,
    'userId': userId,
    'zipCode': zipCode,
    'state': state,
    'city': city,
    'street': street,
    'street2': street2,
    'country': country,
    'createdAt': createdAt.toUtc().toIso8601String(),
    'updatedAt': updatedAt.toUtc().toIso8601String(),
  };
}

/// POST body
class CreateShippingAddressInput {
  CreateShippingAddressInput({
    required this.userId,
    required this.zipCode,
    required this.state,
    required this.city,
    required this.street,
    this.street2,
    this.country = 'JP',
  });

  final String userId;
  final String zipCode;
  final String state;
  final String city;
  final String street;
  final String? street2;
  final String country;

  Map<String, dynamic> toJson() => {
    'userId': userId.trim(),
    'zipCode': zipCode.trim(),
    'state': state.trim(),
    'city': city.trim(),
    'street': street.trim(),
    // backend entity is string, so we send "" when null
    'street2': (street2 ?? '').trim(),
    'country': country.trim(),
  };
}

/// PATCH body (only non-null fields will be sent)
class UpdateShippingAddressInput {
  UpdateShippingAddressInput({
    this.zipCode,
    this.state,
    this.city,
    this.street,
    this.street2,
    this.country,
  });

  final String? zipCode;
  final String? state;
  final String? city;
  final String? street;
  final String? street2;
  final String? country;

  Map<String, dynamic> toJson() {
    final m = <String, dynamic>{};

    void put(String k, String? v) {
      if (v == null) return;
      m[k] = v.trim();
    }

    put('zipCode', zipCode);
    put('state', state);
    put('city', city);
    put('street', street);
    // street2 は「消す」ケースがあり得るので、呼び出し側で "" を渡せば消去扱いにできます
    put('street2', street2);
    put('country', country);

    return m;
  }
}

class ShippingAddressRepositoryHttp {
  ShippingAddressRepositoryHttp({Dio? dio})
    : _dio =
          dio ??
          Dio(
            BaseOptions(
              baseUrl: _resolveApiBase(),
              connectTimeout: const Duration(seconds: 12),
              receiveTimeout: const Duration(seconds: 20),
              headers: const {'Content-Type': 'application/json'},
            ),
          );

  final Dio _dio;
  final CancelToken _cancelToken = CancelToken();

  void dispose() {
    if (!_cancelToken.isCancelled) {
      _cancelToken.cancel('disposed');
    }
  }

  Future<Map<String, String>> _authHeader() async {
    final user = FirebaseAuth.instance.currentUser;
    if (user == null) return {};

    // firebase_auth のバージョン差分で nullable になるケースに備える
    final String? token = await user.getIdToken();
    final t = (token ?? '').trim();
    if (t.isEmpty) return {};

    return {'Authorization': 'Bearer $t'};
  }

  Exception _normalizeDioError(Object e) {
    if (e is DioException) {
      final status = e.response?.statusCode;
      final data = e.response?.data;
      final msg = e.message ?? 'dio_exception';
      return Exception(
        'HTTP error${status != null ? ' ($status)' : ''}: ${data ?? msg}',
      );
    }
    return Exception(e.toString());
  }

  // ------------------------------------------------------------
  // API
  // ------------------------------------------------------------

  /// GET /shipping-addresses/{id}
  Future<ShippingAddress> getById(String id) async {
    final s = id.trim();
    if (s.isEmpty) throw Exception('invalid id');

    try {
      final res = await _dio.get(
        '/shipping-addresses/$s',
        options: Options(headers: await _authHeader()),
        cancelToken: _cancelToken,
      );
      final data = (res.data is Map)
          ? Map<String, dynamic>.from(res.data as Map)
          : <String, dynamic>{};
      return ShippingAddress.fromJson(data);
    } catch (e) {
      throw _normalizeDioError(e);
    }
  }

  /// POST /shipping-addresses
  Future<ShippingAddress> create(CreateShippingAddressInput inData) async {
    try {
      final res = await _dio.post(
        '/shipping-addresses',
        data: inData.toJson(),
        options: Options(headers: await _authHeader()),
        cancelToken: _cancelToken,
      );
      final data = (res.data is Map)
          ? Map<String, dynamic>.from(res.data as Map)
          : <String, dynamic>{};
      return ShippingAddress.fromJson(data);
    } catch (e) {
      throw _normalizeDioError(e);
    }
  }

  /// PATCH /shipping-addresses/{id}
  ///
  /// backend が PUT の場合は `.patch` を `.put` に変更してください。
  Future<ShippingAddress> update(
    String id,
    UpdateShippingAddressInput inData,
  ) async {
    final s = id.trim();
    if (s.isEmpty) throw Exception('invalid id');

    final body = inData.toJson();
    if (body.isEmpty) {
      // no-op: still fetch current
      return getById(s);
    }

    try {
      final res = await _dio.patch(
        '/shipping-addresses/$s',
        data: body,
        options: Options(headers: await _authHeader()),
        cancelToken: _cancelToken,
      );
      final data = (res.data is Map)
          ? Map<String, dynamic>.from(res.data as Map)
          : <String, dynamic>{};
      return ShippingAddress.fromJson(data);
    } catch (e) {
      throw _normalizeDioError(e);
    }
  }

  /// DELETE /shipping-addresses/{id}
  Future<void> delete(String id) async {
    final s = id.trim();
    if (s.isEmpty) throw Exception('invalid id');

    try {
      await _dio.delete(
        '/shipping-addresses/$s',
        options: Options(headers: await _authHeader()),
        cancelToken: _cancelToken,
      );
    } catch (e) {
      throw _normalizeDioError(e);
    }
  }
}
