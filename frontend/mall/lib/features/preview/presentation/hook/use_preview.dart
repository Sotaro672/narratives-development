// frontend/mall/lib/features/preview/presentation/hook/use_preview.dart
import 'dart:convert';

import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http; // ✅ needed for http.Client

import '../../../../app/config/api_base.dart';
import '../../infrastructure/repository.dart';

/// PreviewPage からロジック（data fetch / auth flow / state）を分離するための controller.
/// - Widget 側は「見た目」だけを持つ方針。
/// - FutureBuilder の future と、owner表示用のユーティリティを提供する。
class UsePreviewController {
  UsePreviewController();

  // repositories
  late final PreviewRepositoryHttp _previewRepo;
  late final ScanVerifyRepositoryHttp _scanVerifyRepo;
  late final ScanTransferRepositoryHttp _scanTransferRepo;
  late final MeAvatarRepositoryHttp _meAvatarRepo;

  // args
  String _avatarId = '';
  String _productId = '';
  String? _from;

  // preview future
  Future<MallPreviewResponse?>? _previewFuture;

  // auth flow states
  String? _meAvatarId;
  MallScanVerifyResponse? _verifyResult;
  MallScanTransferResponse? _transferResult;

  bool _busyMe = false;
  bool _busyVerify = false;
  bool _busyTransfer = false;
  bool _transferTriggered = false;

  // init/dispose
  void init({required String avatarId, String? productId, String? from}) {
    _avatarId = avatarId.trim();
    _productId = (productId ?? '').trim();
    _from = from;

    _previewRepo = PreviewRepositoryHttp();
    _scanVerifyRepo = ScanVerifyRepositoryHttp();
    _scanTransferRepo = ScanTransferRepositoryHttp();
    _meAvatarRepo = MeAvatarRepositoryHttp();

    if (_productId.isNotEmpty) {
      _previewFuture = _loadPreview(_productId);
      _kickAuthFlowIfNeeded();
    } else {
      _previewFuture = null;
    }
  }

  void update({required String avatarId, String? productId, String? from}) {
    final newAvatarId = avatarId.trim();
    final newProductId = (productId ?? '').trim();
    final changed =
        newAvatarId != _avatarId || newProductId != _productId || from != _from;

    if (!changed) return;

    _avatarId = newAvatarId;
    _productId = newProductId;
    _from = from;

    // reset state when inputs change
    _meAvatarId = null;
    _verifyResult = null;
    _transferResult = null;

    _busyMe = false;
    _busyVerify = false;
    _busyTransfer = false;
    _transferTriggered = false;

    _previewFuture = _productId.isNotEmpty ? _loadPreview(_productId) : null;

    if (_productId.isNotEmpty) {
      _kickAuthFlowIfNeeded();
    }
  }

  void dispose() {
    _previewRepo.dispose();
    _scanVerifyRepo.dispose();
    _scanTransferRepo.dispose();
    _meAvatarRepo.dispose();
  }

  // getters for View
  String get avatarId => _avatarId;
  String get productId => _productId;
  String? get from => _from;

  Future<MallPreviewResponse?>? get previewFuture => _previewFuture;

  /// FutureBuilder の builder 内で snap.data が型推論されにくい環境向けの補助
  ///
  /// ✅ IMPORTANT:
  /// - Object? を受け取り、MallPreviewResponse のときだけ返す
  /// - それ以外は null
  MallPreviewResponse? previewDataFromSnapshot(Object? v) {
    if (v is MallPreviewResponse) return v;
    return null;
  }

  // ----------------------------
  // helpers
  // ----------------------------
  Future<String> _idTokenOrEmpty(User user) async {
    try {
      final t = await user.getIdToken();
      return (t ?? '').toString();
    } catch (_) {
      return '';
    }
  }

  // ----------------------------
  // Preview
  // ----------------------------
  Future<MallPreviewResponse?> _loadPreview(String productId) async {
    final id = productId.trim();
    if (id.isEmpty) return null;

    final user = FirebaseAuth.instance.currentUser;

    if (user == null) {
      return await _previewRepo.fetchPreviewByProductId(id);
    }

    final token = await _idTokenOrEmpty(user);

    return await _previewRepo.fetchMyPreviewByProductId(
      id,
      headers: {'Authorization': 'Bearer $token'},
    );
  }

  // ----------------------------
  // Auth Flow (me avatar -> verify -> transfer)
  // ----------------------------
  Future<void> _kickAuthFlowIfNeeded() async {
    final productId = _productId;
    if (productId.isEmpty) return;

    final user = FirebaseAuth.instance.currentUser;
    if (user == null) return;

    final current = (_meAvatarId ?? '').trim();
    if (current.isNotEmpty) {
      await _verifyAndMaybeTransfer();
      return;
    }

    await _resolveMeAvatarId();
    await _verifyAndMaybeTransfer();
  }

  Future<void> _resolveMeAvatarId() async {
    if (_busyMe) return;

    final user = FirebaseAuth.instance.currentUser;
    if (user == null) return;

    _busyMe = true;

    try {
      final token = await _idTokenOrEmpty(user);

      final r = await _meAvatarRepo.fetchMeAvatar(
        headers: {'Authorization': 'Bearer $token'},
      );

      final meAvatarId = r.avatarId.trim();
      _meAvatarId = meAvatarId.isEmpty ? null : meAvatarId;
    } catch (_) {
      // ignore (best-effort)
    } finally {
      _busyMe = false;
    }
  }

  Future<void> _verifyAndMaybeTransfer() async {
    final user = FirebaseAuth.instance.currentUser;
    if (user == null) return;

    final productId = _productId.trim();
    final meAvatarId = (_meAvatarId ?? '').trim();
    if (productId.isEmpty || meAvatarId.isEmpty) return;

    if (_verifyResult != null) {
      await _maybeAutoTransfer();
      return;
    }

    if (_busyVerify) return;
    _busyVerify = true;

    try {
      final token = await _idTokenOrEmpty(user);

      final r = await _scanVerifyRepo.verifyScanPurchasedByAvatarId(
        avatarId: meAvatarId,
        productId: productId,
        headers: {'Authorization': 'Bearer $token'},
      );

      _verifyResult = r;

      await _maybeAutoTransfer();
    } catch (_) {
      // ignore (best-effort)
    } finally {
      _busyVerify = false;
    }
  }

  Future<void> _maybeAutoTransfer() async {
    final user = FirebaseAuth.instance.currentUser;
    if (user == null) return;

    final productId = _productId.trim();
    final meAvatarId = (_meAvatarId ?? '').trim();
    final verify = _verifyResult;

    if (productId.isEmpty || meAvatarId.isEmpty || verify == null) return;
    if (!verify.matched) return;

    if (_transferTriggered || _transferResult != null || _busyTransfer) return;
    _transferTriggered = true;

    _busyTransfer = true;

    try {
      final token = await _idTokenOrEmpty(user);

      final r = await _scanTransferRepo.transferScanPurchased(
        productId: productId,
        headers: {'Authorization': 'Bearer $token'},
      );

      _transferResult = r;
    } catch (_) {
      // ignore (best-effort)
    } finally {
      _busyTransfer = false;
    }
  }

  // ----------------------------
  // Owner label (view helper)
  // ----------------------------
  String ownerLabel(MallOwnerInfo? owner) {
    if (owner == null) return '-';

    final avatarName = owner.avatarName.trim();
    final brandName = owner.brandName.trim();
    final avatarId = owner.avatarId.trim();
    final brandId = owner.brandId.trim();

    if (avatarName.isNotEmpty) return avatarName;
    if (brandName.isNotEmpty) return brandName;

    if (avatarId.isNotEmpty) return avatarId;
    if (brandId.isNotEmpty) return brandId;

    return '-';
  }
}

/// /mall/me/avatar 用（このファイル内で完結させるための最小実装）
///
/// NOTE:
/// - 元の preview.dart 末尾にあった実装を hook に移設。
class MeAvatarRepositoryHttp {
  MeAvatarRepositoryHttp({http.Client? client})
    : _client = client ?? http.Client();

  final http.Client _client;

  void dispose() {
    _client.close();
  }

  /// GET /mall/me/avatar
  Future<MallOwnerInfo> fetchMeAvatar({
    String? baseUrl,
    Map<String, String>? headers,
  }) async {
    final base = (baseUrl ?? '').trim();
    final resolvedBase = base.isNotEmpty ? base : resolveApiBase();

    final b = normalizeBaseUrl(resolvedBase);
    final uri = Uri.parse('$b/mall/me/avatar');

    final mergedHeaders = <String, String>{...jsonHeaders()};
    if (headers != null) {
      mergedHeaders.addAll(headers);
    }

    final auth = (mergedHeaders['Authorization'] ?? '').trim();
    if (auth.isEmpty) {
      throw ArgumentError(
        'Authorization header is required for /mall/me/avatar',
      );
    }

    final res = await _client.get(uri, headers: mergedHeaders);

    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw HttpException(
        'fetchMeAvatar failed: ${res.statusCode}',
        url: uri.toString(),
        body: res.body,
      );
    }

    final decoded = jsonDecode(res.body);
    if (decoded is! Map) {
      throw const FormatException('invalid json shape (expected object)');
    }

    return MallOwnerInfo.fromJson(decoded.cast<String, dynamic>());
  }
}

/// Minimal HTTP exception (kept local to avoid extra deps).
class HttpException implements Exception {
  HttpException(this.message, {this.url, this.body});

  final String message;
  final String? url;
  final String? body;

  @override
  String toString() {
    final u = (url ?? '').trim();
    final b = (body ?? '').trim();
    if (u.isEmpty && b.isEmpty) return message;
    if (b.isEmpty) return '$message ($u)';
    return '$message ($u) body=$b';
  }
}
