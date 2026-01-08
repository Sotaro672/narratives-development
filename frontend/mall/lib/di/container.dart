// frontend/mall/lib/di/container.dart
//
// Mall app DI container (minimal, feature-first).
// - Centralizes API_BASE resolution (with fallback)
// - Creates repositories/services with shared FirebaseAuth + http.Client
// - Provides a single place to dispose resources
//
// NOTE:
// Your repositories currently add their own Dio interceptors internally.
// To avoid duplicate interceptors, this container does NOT share a single Dio instance.
// Instead it shares baseUrl/auth and lets each repository manage its own Dio safely.

import 'package:flutter/foundation.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http;

import '../features/user/infrastructure/user_repository_http.dart';
import '../features/shippingAddress/infrastructure/shipping_address_repository_http.dart';
import '../features/auth/application/shipping_address_service.dart';

class AppContainer {
  AppContainer._({
    required String apiBase,
    FirebaseAuth? auth,
    http.Client? httpClient,
    UserRepositoryHttp? userRepo,
    ShippingAddressRepositoryHttp? shippingRepo,
    ShippingAddressService? shippingService,
  }) : apiBase = _normalizeBaseUrl(_ensureSnsBase(apiBase)),
       auth = auth ?? FirebaseAuth.instance,
       httpClient = httpClient ?? http.Client(),
       _userRepoOverride = userRepo,
       _shippingRepoOverride = shippingRepo,
       _shippingServiceOverride = shippingService;

  // ----------------------------
  // Singleton
  // ----------------------------
  static AppContainer? _instance;

  static AppContainer get I {
    return _instance ??= AppContainer._(apiBase: _resolveApiBase());
  }

  /// For tests / local experiments: replace the singleton.
  static void setInstance(AppContainer container) {
    _instance = container;
  }

  /// Dispose and clear singleton (useful in tests).
  static void disposeInstance() {
    _instance?.dispose();
    _instance = null;
  }

  // ----------------------------
  // Public deps
  // ----------------------------
  final String apiBase; // <-- ends with /sns
  final FirebaseAuth auth;
  final http.Client httpClient;

  // ----------------------------
  // Overrides (for tests)
  // ----------------------------
  final UserRepositoryHttp? _userRepoOverride;
  final ShippingAddressRepositoryHttp? _shippingRepoOverride;
  final ShippingAddressService? _shippingServiceOverride;

  // ----------------------------
  // Lazy singletons
  // ----------------------------
  UserRepositoryHttp? _userRepo;
  ShippingAddressRepositoryHttp? _shippingRepo;
  ShippingAddressService? _shippingService;

  UserRepositoryHttp get userRepository {
    return _userRepo ??=
        _userRepoOverride ?? UserRepositoryHttp(auth: auth, baseUrl: apiBase);
  }

  ShippingAddressRepositoryHttp get shippingAddressRepository {
    return _shippingRepo ??=
        _shippingRepoOverride ??
        ShippingAddressRepositoryHttp(auth: auth, baseUrl: apiBase);
  }

  ShippingAddressService get shippingAddressService {
    return _shippingService ??=
        _shippingServiceOverride ??
        ShippingAddressService(
          auth: auth,
          httpClient: httpClient,
          userRepo: userRepository,
          shipRepo: shippingAddressRepository,
          baseUrl: apiBase,
        );
  }

  // ----------------------------
  // Lifecycle
  // ----------------------------
  bool _disposed = false;

  void dispose() {
    if (_disposed) return;
    _disposed = true;

    try {
      _shippingService?.dispose();
    } catch (_) {}
    try {
      _userRepo?.dispose();
    } catch (_) {}
    try {
      _shippingRepo?.dispose();
    } catch (_) {}
    try {
      httpClient.close();
    } catch (_) {}
  }

  // ----------------------------
  // API_BASE resolution
  // ----------------------------

  // ✅ Mall app should talk to /mall/* endpoints.
  static const String _fallbackBaseUrl =
      'https://narratives-backend-871263659099.asia-northeast1.run.app/mall';

  static String _resolveApiBase() {
    const fromDefine = String.fromEnvironment('API_BASE', defaultValue: '');
    final v = fromDefine.trim();
    if (v.isNotEmpty) return _normalizeBaseUrl(_ensureSnsBase(v));

    return _normalizeBaseUrl(_fallbackBaseUrl);
  }

  static String _normalizeBaseUrl(String v) {
    final s = v.trim();
    if (s.isEmpty) return '';
    return s.endsWith('/') ? s.substring(0, s.length - 1) : s;
  }

  /// Ensure base URL ends with "/sns".
  /// - If user passes root "...run.app", we convert to "...run.app/sns".
  /// - If user already passes ".../sns", keep as-is.
  static String _ensureSnsBase(String base) {
    final b = _normalizeBaseUrl(base);
    if (b.isEmpty) return b;
    if (b.endsWith('/sns')) return b;
    return '$b/sns';
  }

  // ----------------------------
  // Debug helper
  // ----------------------------
  static void debugPrintSummary() {
    if (!kDebugMode) return;
    final c = AppContainer.I;
    debugPrint('[AppContainer] apiBase="${c.apiBase}"');
  }
}
