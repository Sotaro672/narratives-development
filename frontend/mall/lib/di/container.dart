// frontend/mall/lib/di/container.dart

import 'package:flutter/foundation.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http;

import '../app/config/api_base.dart';

import '../features/user/infrastructure/user_repository_http.dart';
import '../features/shippingAddress/infrastructure/repository_http.dart';
import '../features/auth/application/shipping_address_service.dart';

// ✅ ADD: avatar repo
import '../features/avatar/infrastructure/repository_http.dart';

class AppContainer {
  AppContainer._({
    required String apiBase,
    FirebaseAuth? auth,
    http.Client? httpClient,
    UserRepositoryHttp? userRepo,
    ShippingAddressRepositoryHttp? shippingRepo,
    ShippingAddressService? shippingService,

    // ✅ ADD (override for tests)
    AvatarRepositoryHttp? avatarRepo,
  }) : apiBase = _normalizeBaseUrl(apiBase), // ✅ ROOT ONLY (no /mall)
       auth = auth ?? FirebaseAuth.instance,
       httpClient = httpClient ?? http.Client(),
       _userRepoOverride = userRepo,
       _shippingRepoOverride = shippingRepo,
       _shippingServiceOverride = shippingService,
       _avatarRepoOverride = avatarRepo;

  // ----------------------------
  // Singleton
  // ----------------------------
  static AppContainer? _instance;

  static AppContainer get I {
    return _instance ??= AppContainer._(apiBase: _resolveApiBaseRoot());
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

  /// ✅ ROOT base (never ends with /mall)
  /// Example: https://...run.app
  final String apiBase;

  /// ✅ Convenience getter if some legacy code really needs "/mall" base.
  /// Prefer passing apiBase to repos/services and let them append "/mall/..."
  String get apiMallBase => _ensureSuffix(apiBase, '/mall');

  final FirebaseAuth auth;
  final http.Client httpClient;

  // ----------------------------
  // Overrides (for tests)
  // ----------------------------
  final UserRepositoryHttp? _userRepoOverride;
  final ShippingAddressRepositoryHttp? _shippingRepoOverride;
  final ShippingAddressService? _shippingServiceOverride;

  // ✅ ADD
  final AvatarRepositoryHttp? _avatarRepoOverride;

  // ----------------------------
  // Lazy singletons
  // ----------------------------
  UserRepositoryHttp? _userRepo;
  ShippingAddressRepositoryHttp? _shippingRepo;
  ShippingAddressService? _shippingService;

  // ✅ ADD
  AvatarRepositoryHttp? _avatarRepo;

  /// ✅ IMPORTANT:
  /// - baseUrl は ROOT を渡す（repo 側で "/mall/..." を付ける）
  /// - ここで "/mall" を付けると repo 側と二重になり得る
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
          baseUrl: apiBase, // ✅ ROOT
        );
  }

  /// ✅ ADD: Avatar repository (Mall avatar endpoints)
  ///
  /// IMPORTANT:
  /// - AvatarRepositoryHttp 側が "/mall/..." を付けて叩く想定なら、ROOT を渡すのが正解。
  /// - これで "/mall/mall" を根絶できる。
  AvatarRepositoryHttp get avatarRepositoryHttp {
    return _avatarRepo ??=
        _avatarRepoOverride ??
        AvatarRepositoryHttp(auth: auth, baseUrl: apiBase);
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

    // ✅ ADD
    try {
      _avatarRepo?.dispose();
    } catch (_) {}

    try {
      httpClient.close();
    } catch (_) {}
  }

  // ----------------------------
  // API_BASE resolution
  // ----------------------------

  /// ✅ single source of truth: app/config/api_base.dart
  /// Returns ROOT (no /mall, no /sns).
  static String _resolveApiBaseRoot() {
    final root = resolveApiBase(); // e.g. https://...run.app
    return _normalizeBaseUrl(root);
  }

  static String _normalizeBaseUrl(String v) {
    final s = v.trim();
    if (s.isEmpty) return '';
    return s.replaceAll(RegExp(r'\/+$'), ''); // remove trailing slashes
  }

  static String _ensureSuffix(String base, String suffix) {
    final b = _normalizeBaseUrl(base);
    if (b.isEmpty) return b;
    if (b.endsWith(suffix)) return b;
    return '$b$suffix';
  }

  // ----------------------------
  // Debug helper
  // ----------------------------
  static void debugPrintSummary() {
    if (!kDebugMode) return;
    final c = AppContainer.I;
    debugPrint(
      '[AppContainer] apiBase="${c.apiBase}" apiMallBase="${c.apiMallBase}"',
    );
  }
}
