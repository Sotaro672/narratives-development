// frontend/sns/lib/features/payment/presentation/page/payment.dart
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;

// ✅ Firebase: uid -> avatarId を Firestore から解決する
import 'package:firebase_auth/firebase_auth.dart';
import 'package:cloud_firestore/cloud_firestore.dart';

// ✅ API_BASE 解決ロジックを既存と揃える（cart と同様）
import '../../../inventory/infrastructure/inventory_repository_http.dart';

class PaymentPage extends StatefulWidget {
  const PaymentPage({super.key, required this.avatarId, this.from});

  final String avatarId;
  final String? from;

  @override
  State<PaymentPage> createState() => _PaymentPageState();
}

class _PaymentPageState extends State<PaymentPage> {
  late final PaymentRepositoryHttp _repo;
  late Future<PaymentContextDTO> _future;

  bool _busy = false;

  // ✅ 解決済み avatarId を保持（Header 表示/デバッグにも使える）
  String _resolvedAvatarId = '';

  String get _incomingAvatarId => widget.avatarId.trim();

  @override
  void initState() {
    super.initState();
    _repo = PaymentRepositoryHttp();
    _kickLoad();
  }

  @override
  void dispose() {
    _repo.dispose();
    super.dispose();
  }

  void _kickLoad() {
    _future = _load();
  }

  Future<PaymentContextDTO> _load() async {
    final aid = await _resolveAvatarIdForRequest();
    _resolvedAvatarId = aid;
    return _repo.fetchPaymentContext(avatarId: aid);
  }

  /// ✅ avatarId が uid だった場合でも、Firestore から avatarId を確定して返す
  Future<String> _resolveAvatarIdForRequest() async {
    final incoming = _incomingAvatarId;
    if (incoming.isEmpty) throw ArgumentError('avatarId is required');

    final uid = (FirebaseAuth.instance.currentUser?.uid ?? '').trim();

    // 1) 期待通り avatarId が来ているケース（uid と一致しないならそのまま採用）
    //    ※「たまたま uid と同じ文字列の avatarId」が存在する設計はしない前提
    if (uid.isEmpty) {
      // 未ログインなら、渡された avatarId を信じるしかない
      return incoming;
    }
    if (incoming != uid) {
      return incoming;
    }

    // 2) incoming が uid だった → Firestore で avatars where userId==uid を引いて doc.id を avatarId とする
    final qs = await FirebaseFirestore.instance
        .collection('avatars')
        .where('userId', isEqualTo: uid)
        .limit(1)
        .get();

    if (qs.docs.isEmpty) {
      throw StateError('No avatar found for userId=$uid');
    }

    final avatarId = qs.docs.first.id.trim();
    if (avatarId.isEmpty) {
      throw StateError('Resolved avatarId is empty for userId=$uid');
    }
    return avatarId;
  }

  Future<void> _reload() async {
    setState(() {
      _kickLoad();
    });
  }

  Future<void> _withBusy(Future<void> Function() fn) async {
    if (_busy) return;
    setState(() => _busy = true);
    try {
      await fn();
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  // ✅ ここでは「購入確定のAPI起票」はまだ実装しない（次工程）
  Future<void> _confirmPurchase(PaymentContextDTO ctx) async {
    await _withBusy(() async {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('購入確定は次工程で実装します（UIは準備できています）')),
      );
    });
  }

  @override
  Widget build(BuildContext context) {
    if (_incomingAvatarId.isEmpty) {
      return const Scaffold(body: Center(child: Text('avatarId is required')));
    }

    return Scaffold(
      appBar: AppBar(
        title: const Text('Payment'),
        leading: IconButton(
          tooltip: 'Back',
          icon: const Icon(Icons.arrow_back),
          onPressed: () => Navigator.of(context).maybePop(),
        ),
        actions: [
          IconButton(
            tooltip: 'Reload',
            onPressed: _reload,
            icon: const Icon(Icons.refresh),
          ),
        ],
      ),
      body: Stack(
        children: [
          FutureBuilder<PaymentContextDTO>(
            future: _future,
            builder: (context, snap) {
              final isLoading =
                  snap.connectionState == ConnectionState.waiting &&
                  !snap.hasData;

              if (isLoading) {
                return const Center(child: CircularProgressIndicator());
              }

              if (snap.hasError) {
                return _ErrorView(
                  errorText: snap.error.toString(),
                  onRetry: _reload,
                );
              }

              final ctx = snap.data;
              if (ctx == null) {
                return _ErrorView(errorText: 'No data', onRetry: _reload);
              }

              final canConfirm =
                  ctx.userId.trim().isNotEmpty &&
                  (ctx.shippingAddress?.isNotEmpty ?? false) &&
                  (ctx.billingAddress?.isNotEmpty ?? false);

              return SingleChildScrollView(
                padding: const EdgeInsets.fromLTRB(12, 12, 12, 24),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    // ✅ 解決済み avatarId を表示（incoming が uid でも、ここは avatarId になる）
                    _HeaderCard(
                      avatarId: ctx.avatarId,
                      userId: ctx.userId,
                      resolvedAvatarId: _resolvedAvatarId,
                      incomingAvatarId: _incomingAvatarId,
                    ),
                    const SizedBox(height: 12),

                    _AddressCard(
                      title: 'Shipping Address',
                      address: ctx.shippingAddress,
                      emptyText: '配送先住所が未登録です',
                    ),
                    const SizedBox(height: 12),

                    _AddressCard(
                      title: 'Billing Address',
                      address: ctx.billingAddress,
                      emptyText: '請求先住所が未登録です',
                    ),
                    const SizedBox(height: 16),

                    SizedBox(
                      height: 48,
                      child: FilledButton(
                        onPressed: canConfirm
                            ? () => _confirmPurchase(ctx)
                            : null,
                        child: const Text('購入を確定する'),
                      ),
                    ),
                    const SizedBox(height: 8),
                    Text(
                      canConfirm
                          ? '※ 次工程で order/payment 起票を実装します'
                          : '※ 住所が揃うと購入確定できます（次工程で住所登録導線も整備）',
                      style: Theme.of(context).textTheme.bodySmall,
                      textAlign: TextAlign.center,
                    ),
                  ],
                ),
              );
            },
          ),
          if (_busy)
            Positioned.fill(
              child: IgnorePointer(
                ignoring: true,
                child: Container(
                  color: Colors.black.withValues(alpha: 0.06),
                  child: const Center(child: CircularProgressIndicator()),
                ),
              ),
            ),
        ],
      ),
    );
  }
}

// ============================================================
// Repository (PaymentContext: avatarId -> userId -> addresses)
// ============================================================

class PaymentRepositoryHttp {
  PaymentRepositoryHttp({http.Client? client, String? apiBase})
    : _client = client ?? http.Client(),
      _apiBase = (apiBase ?? const String.fromEnvironment('API_BASE')).trim();

  final http.Client _client;

  /// Optional override. If empty, resolveSnsApiBase() will be used.
  final String _apiBase;

  void dispose() {
    _client.close();
  }

  Future<PaymentContextDTO> fetchPaymentContext({
    required String avatarId,
  }) async {
    final aid = avatarId.trim();
    if (aid.isEmpty) throw ArgumentError('avatarId is required');

    // 1) avatarId -> userId
    final avatar = await _fetchAvatar(aid);
    final userId = (avatar.userId).trim();
    if (userId.isEmpty) {
      return PaymentContextDTO(
        avatarId: aid,
        userId: '',
        shippingAddress: null,
        billingAddress: null,
      );
    }

    // 2) userId -> addresses（複数候補エンドポイントを順に試す）
    final shipping = await _fetchAddressFlexible(
      kind: _AddressKind.shipping,
      userId: userId,
    );
    final billing = await _fetchAddressFlexible(
      kind: _AddressKind.billing,
      userId: userId,
    );

    return PaymentContextDTO(
      avatarId: aid,
      userId: userId,
      shippingAddress: shipping,
      billingAddress: billing,
    );
  }

  Future<AvatarDTO> _fetchAvatar(String avatarId) async {
    // try 1: /sns/avatars/{avatarId}
    try {
      final uri = _uri('/sns/avatars/$avatarId');
      final res = await _client.get(uri, headers: _headersJson());
      if (res.statusCode >= 200 && res.statusCode < 300) {
        final m = _decodeJsonMap(res.body);
        return AvatarDTO.fromJson(m, fallbackAvatarId: avatarId);
      }
    } catch (_) {}

    // try 2: /sns/avatar?avatarId=...
    final uri2 = _uri('/sns/avatar', qp: {'avatarId': avatarId});
    final res2 = await _client.get(uri2, headers: _headersJson());
    if (res2.statusCode >= 200 && res2.statusCode < 300) {
      final m = _decodeJsonMap(res2.body);
      return AvatarDTO.fromJson(m, fallbackAvatarId: avatarId);
    }

    _throwHttpError(res2);
    throw StateError('unreachable');
  }

  Future<Map<String, dynamic>?> _fetchAddressFlexible({
    required _AddressKind kind,
    required String userId,
  }) async {
    try {
      final uri = _uri('/sns/users/$userId');
      final res = await _client.get(uri, headers: _headersJson());
      if (res.statusCode >= 200 && res.statusCode < 300) {
        final m = _decodeJsonMap(res.body);
        final picked = _pickAddressFromUserDoc(m, kind: kind);
        if (picked != null && picked.isNotEmpty) return picked;
      }
    } catch (_) {}

    final path = (kind == _AddressKind.shipping)
        ? '/sns/shipping-addresses'
        : '/sns/billing-addresses';

    try {
      final uri = _uri(path, qp: {'userId': userId});
      final res = await _client.get(uri, headers: _headersJson());
      if (res.statusCode >= 200 && res.statusCode < 300) {
        final m = _decodeJsonMap(res.body);
        final picked = _pickAddressFromAddressResponse(m);
        if (picked != null && picked.isNotEmpty) return picked;
      }
    } catch (_) {}

    try {
      final uri = _uri('$path/$userId');
      final res = await _client.get(uri, headers: _headersJson());
      if (res.statusCode >= 200 && res.statusCode < 300) {
        final m = _decodeJsonMap(res.body);
        final picked = _pickAddressFromAddressResponse(m);
        if (picked != null && picked.isNotEmpty) return picked;
      }
    } catch (_) {}

    return null;
  }

  Map<String, dynamic>? _pickAddressFromUserDoc(
    Map<String, dynamic> userDoc, {
    required _AddressKind kind,
  }) {
    final keys = (kind == _AddressKind.shipping)
        ? const ['shippingAddress', 'shipping_address', 'shipping', 'ship']
        : const ['billingAddress', 'billing_address', 'billing', 'bill'];

    for (final k in keys) {
      final v = userDoc[k];
      if (v is Map<String, dynamic>) return v;
      if (v is Map) return v.cast<String, dynamic>();
    }
    return null;
  }

  Map<String, dynamic>? _pickAddressFromAddressResponse(
    Map<String, dynamic> res,
  ) {
    final v = res['address'];
    if (v is Map<String, dynamic>) return v;
    if (v is Map) return v.cast<String, dynamic>();

    final d = res['data'];
    if (d is Map<String, dynamic>) return d;
    if (d is Map) return d.cast<String, dynamic>();

    return res;
  }

  Map<String, dynamic> _decodeJsonMap(String body) {
    final raw = body.trim().isEmpty ? '{}' : body;
    final v = jsonDecode(raw);
    if (v is Map<String, dynamic>) return v;
    if (v is Map) return v.cast<String, dynamic>();
    throw const FormatException('invalid json response');
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
    throw PaymentHttpException(statusCode: status, message: msg);
  }

  Uri _uri(String path, {Map<String, String>? qp}) {
    final base = (_apiBase.isNotEmpty ? _apiBase : resolveSnsApiBase()).trim();

    if (base.isEmpty) {
      throw StateError(
        'API_BASE is not set (use --dart-define=API_BASE=https://...)',
      );
    }

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

  Map<String, String> _headersJson() => const {
    'Content-Type': 'application/json; charset=utf-8',
    'Accept': 'application/json',
  };
}

enum _AddressKind { shipping, billing }

// ============================================================
// DTOs
// ============================================================

class PaymentContextDTO {
  PaymentContextDTO({
    required this.avatarId,
    required this.userId,
    required this.shippingAddress,
    required this.billingAddress,
  });

  final String avatarId;
  final String userId;

  final Map<String, dynamic>? shippingAddress;
  final Map<String, dynamic>? billingAddress;
}

class AvatarDTO {
  AvatarDTO({required this.avatarId, required this.userId});

  final String avatarId;
  final String userId;

  factory AvatarDTO.fromJson(
    Map<String, dynamic> json, {
    required String fallbackAvatarId,
  }) {
    final aid = (json['avatarId'] ?? json['id'] ?? fallbackAvatarId)
        .toString()
        .trim();

    final uid =
        (json['userId'] ??
                json['uid'] ??
                json['memberId'] ??
                json['ownerUserId'] ??
                '')
            .toString()
            .trim();

    return AvatarDTO(avatarId: aid, userId: uid);
  }
}

class PaymentHttpException implements Exception {
  PaymentHttpException({required this.statusCode, required this.message});

  final int statusCode;
  final String message;

  @override
  String toString() =>
      'PaymentHttpException(statusCode=$statusCode, message=$message)';
}

// ============================================================
// UI parts
// ============================================================

class _HeaderCard extends StatelessWidget {
  const _HeaderCard({
    required this.avatarId,
    required this.userId,
    required this.resolvedAvatarId,
    required this.incomingAvatarId,
  });

  final String avatarId;
  final String userId;

  // ✅ デバッグ用に表示（混入確認）
  final String resolvedAvatarId;
  final String incomingAvatarId;

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).textTheme;

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('確認', style: t.titleMedium),
            const SizedBox(height: 8),
            Text('incoming avatarId: $incomingAvatarId', style: t.bodySmall),
            const SizedBox(height: 4),
            Text('resolved avatarId: $resolvedAvatarId', style: t.bodySmall),
            const SizedBox(height: 4),
            Text('ctx.avatarId: $avatarId', style: t.bodySmall),
            const SizedBox(height: 4),
            Text(
              userId.trim().isEmpty
                  ? 'userId: (not resolved)'
                  : 'userId: $userId',
              style: t.bodySmall,
            ),
          ],
        ),
      ),
    );
  }
}

class _AddressCard extends StatelessWidget {
  const _AddressCard({
    required this.title,
    required this.address,
    required this.emptyText,
  });

  final String title;
  final Map<String, dynamic>? address;
  final String emptyText;

  String _s(dynamic v) => (v ?? '').toString().trim();

  List<MapEntry<String, String>> _toPairs(Map<String, dynamic> m) {
    final preferredKeys = <String>[
      'fullName',
      'name',
      'phone',
      'email',
      'postalCode',
      'zip',
      'prefecture',
      'state',
      'city',
      'address1',
      'address2',
      'line1',
      'line2',
      'country',
    ];

    final used = <String>{};
    final pairs = <MapEntry<String, String>>[];

    for (final k in preferredKeys) {
      if (!m.containsKey(k)) continue;
      final v = _s(m[k]);
      if (v.isEmpty) continue;
      used.add(k);
      pairs.add(MapEntry(k, v));
    }

    for (final e in m.entries) {
      final k = e.key.toString();
      if (used.contains(k)) continue;
      final v = _s(e.value);
      if (v.isEmpty) continue;
      pairs.add(MapEntry(k, v));
      if (pairs.length >= 18) break;
    }

    return pairs;
  }

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).textTheme;
    final m = address;

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(title, style: t.titleMedium),
            const SizedBox(height: 10),
            if (m == null || m.isEmpty)
              Text(emptyText, style: t.bodyMedium)
            else
              ..._toPairs(m).map(
                (e) => Padding(
                  padding: const EdgeInsets.only(bottom: 6),
                  child: Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      SizedBox(
                        width: 110,
                        child: Text(
                          e.key,
                          style: t.bodySmall?.copyWith(
                            color: Theme.of(context).textTheme.bodySmall?.color
                                ?.withValues(alpha: 0.7),
                          ),
                        ),
                      ),
                      Expanded(child: Text(e.value, style: t.bodySmall)),
                    ],
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }
}

class _ErrorView extends StatelessWidget {
  const _ErrorView({required this.errorText, required this.onRetry});

  final String errorText;
  final Future<void> Function() onRetry;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(18),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Text('Error'),
            const SizedBox(height: 8),
            Text(errorText, textAlign: TextAlign.center),
            const SizedBox(height: 12),
            OutlinedButton(
              onPressed: () => onRetry(),
              child: const Text('Retry'),
            ),
          ],
        ),
      ),
    );
  }
}
