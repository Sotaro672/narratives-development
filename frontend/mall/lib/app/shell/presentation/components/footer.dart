// frontend\mall\lib\app\shell\presentation\components\footer.dart
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:mobile_scanner/mobile_scanner.dart';

import '../state/catalog_selection_store.dart';

// ✅ NEW: 3 buttons extracted
import 'footer_buttons.dart';

// ✅ routing constants
import '../../../routing/routes.dart';

/// Minimal footer widget (layout primitive).
class AppFooter extends StatelessWidget {
  const AppFooter({
    super.key,
    this.left,
    this.center,
    this.right,
    this.padding = const EdgeInsets.symmetric(horizontal: 12, vertical: 12),
  });

  final Widget? left;
  final Widget? center;
  final Widget? right;

  final EdgeInsets padding;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: Theme.of(context).cardColor,
      elevation: 0,
      child: SafeArea(
        top: false,
        child: Padding(
          padding: padding,
          child: Row(
            children: [
              SizedBox(
                width: 96,
                child: Align(alignment: Alignment.centerLeft, child: left),
              ),
              Expanded(child: Center(child: center)),
              SizedBox(
                width: 96,
                child: Align(alignment: Alignment.centerRight, child: right),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// Signed-in footer (Shop / Scan(or AddToCart) / AvatarIcon)
/// - Shows ONLY when FirebaseAuth.currentUser != null
class SignedInFooter extends StatelessWidget {
  const SignedInFooter({super.key});

  bool _isCatalogPath(BuildContext context) {
    final path = GoRouterState.of(context).uri.path;
    return path.startsWith('/catalog/');
  }

  bool _isCartPath(BuildContext context) {
    final path = GoRouterState.of(context).uri.path;
    return path == '/cart';
  }

  bool _isPaymentPath(BuildContext context) {
    final path = GoRouterState.of(context).uri.path;
    return path == '/payment';
  }

  String _currentUriString(BuildContext context) {
    return GoRouterState.of(context).uri.toString();
  }

  String _catalogListIdOrEmpty(BuildContext context) {
    final path = GoRouterState.of(context).uri.path; // /catalog/:listId
    final parts = path.split('/');
    if (parts.length >= 3 && parts[1] == 'catalog') return parts[2];
    return '';
  }

  String _currentAvatarIdOrEmpty(BuildContext context) {
    final qp = GoRouterState.of(context).uri.queryParameters;
    return (qp[AppQueryKey.avatarId] ?? '').trim();
  }

  /// ✅ 現在URLの avatarId を必要に応じて引き継ぐ
  Uri _buildUriPreserveAvatarId(
    BuildContext context,
    String path, {
    Map<String, String>? qp,
  }) {
    final current = GoRouterState.of(context).uri;
    final merged = <String, String>{...(qp ?? {})};

    // keep avatarId if not explicitly provided
    if (!merged.containsKey(AppQueryKey.avatarId)) {
      final aid = (current.queryParameters[AppQueryKey.avatarId] ?? '').trim();
      if (aid.isNotEmpty) merged[AppQueryKey.avatarId] = aid;
    }

    return Uri(path: path, queryParameters: merged.isEmpty ? null : merged);
  }

  /// ------------------------------------------------------------
  /// ✅ `from` は URL で壊れやすい（Hash + `?` `&` 混在）ので base64url で安全に運ぶ
  String _encodeFrom(String raw) {
    final s = raw.trim();
    if (s.isEmpty) return '';
    return base64UrlEncode(utf8.encode(s));
  }

  /// ------------------------------------------------------------
  /// ✅ /:productId が固定パスと衝突しないように除外（scan時の安全弁）
  bool _isReservedTopSegment(String seg) {
    const reserved = <String>{
      'login',
      'create-account',
      'shipping-address',
      'billing-address',
      'avatar-create',
      'avatar-edit',
      'avatar',
      'user-edit',
      'cart',
      'preview',
      'payment',
      'catalog',
    };
    return reserved.contains(seg);
  }

  /// ------------------------------------------------------------
  /// ✅ QRでスキャンした文字列を「アプリ内遷移先URI」に正規化して返す
  ///
  /// 想定入力：
  /// - https://narratives.jp/{productId}
  /// - https://narratives.jp/preview?productId=...
  /// - /preview?productId=...
  /// - {productId}（生IDだけ）
  Uri? _normalizeScannedToAppUri(
    String raw, {
    required String currentFrom,
    required String avatarId,
  }) {
    final s = raw.trim();
    if (s.isEmpty) return null;

    Uri? u;

    // 1) まず Uri として解釈
    try {
      u = Uri.parse(s);
    } catch (_) {
      u = null;
    }

    // 2) scheme が無い & / でもない場合は「生 productId」とみなす
    if (u == null || (u.scheme.isEmpty && !s.startsWith('/'))) {
      final pid = s;
      if (pid.isEmpty) return null;
      if (_isReservedTopSegment(pid)) return null;

      final qp = <String, String>{
        if (avatarId.trim().isNotEmpty) AppQueryKey.avatarId: avatarId.trim(),
        // どこから来たか（戻る用）
        AppQueryKey.from: _encodeFrom(currentFrom),
      };
      return Uri(path: '/$pid', queryParameters: qp);
    }

    // 3) https://... の場合は path/query だけ抽出してアプリ内遷移にする
    final isHttp = u.scheme == 'http' || u.scheme == 'https';
    final path = (isHttp ? u.path : u.path).trim().isEmpty ? '/' : u.path;

    // fragment はルーティングを壊しやすいので捨てる
    final extracted = Uri(
      path: path.startsWith('/') ? path : '/$path',
      queryParameters: u.queryParameters.isEmpty ? null : u.queryParameters,
    );

    // 4) パスがトップ1階層（= /{something}）の場合、reserved は弾く
    final segs = extracted.pathSegments;
    if (segs.length == 1) {
      final top = (segs.first).trim();
      if (top.isNotEmpty && _isReservedTopSegment(top)) {
        return null;
      }
    }

    // 5) avatarId / from を付与（既に付いていれば尊重）
    final merged = <String, String>{...extracted.queryParameters};

    if (!merged.containsKey(AppQueryKey.avatarId) &&
        avatarId.trim().isNotEmpty) {
      merged[AppQueryKey.avatarId] = avatarId.trim();
    }

    // from は「現在画面」を入れる（PreviewPageの戻り用途）
    if (!merged.containsKey(AppQueryKey.from)) {
      merged[AppQueryKey.from] = _encodeFrom(currentFrom);
    }

    return extracted.replace(queryParameters: merged.isEmpty ? null : merged);
  }

  @override
  Widget build(BuildContext context) {
    return StreamBuilder<User?>(
      // ✅ photoURL/displayName 更新も拾う
      stream: FirebaseAuth.instance.userChanges(),
      builder: (context, snap) {
        final user = FirebaseAuth.instance.currentUser ?? snap.data;

        if (user == null) return const SizedBox.shrink();

        final isCatalog = _isCatalogPath(context);
        final isCart = _isCartPath(context);
        final isPayment = _isPaymentPath(context);

        final from = _currentUriString(context);
        final listId = isCatalog ? _catalogListIdOrEmpty(context) : '';

        // ✅ いまURLに avatarId があれば引き継ぐ（footer 遷移で消さない）
        final avatarId = _currentAvatarIdOrEmpty(context);

        return Material(
          color: Theme.of(context).cardColor,
          elevation: 0,
          child: SafeArea(
            top: false,
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 10),
              child: Row(
                children: [
                  _FooterItem(
                    icon: Icons.storefront_outlined,
                    onTap: () {
                      final uri = _buildUriPreserveAvatarId(context, '/');
                      context.go(uri.toString());
                    },
                  ),
                  const SizedBox(width: 8),

                  // ✅ 中央：
                  // - catalog: カートに入れる
                  // - cart: 購入する（paymentへ）
                  // - payment: 支払を確定する
                  // - other: Scan（スキャンしたURLへ遷移＝Previewへ）
                  Expanded(
                    child: Center(
                      child: isCatalog
                          ? ValueListenableBuilder<CatalogSelection>(
                              valueListenable: CatalogSelectionStore.notifier,
                              builder: (context, sel, _) {
                                final sameList =
                                    sel.listId.trim() == listId.trim();

                                final mid = (sel.modelId ?? '').trim();
                                final stock = sel.stockCount ?? 0;

                                // ✅ backend が必須にした inventoryId/listId を揃える
                                final invId = sel.inventoryId.trim();

                                // ✅ 在庫 0 は押下不可 + 必須IDが揃っていること
                                final enabled =
                                    sameList &&
                                    invId.isNotEmpty &&
                                    listId.trim().isNotEmpty &&
                                    mid.isNotEmpty &&
                                    stock > 0;

                                return AddToCartButton(
                                  from: from,
                                  inventoryId: invId,
                                  listId: listId,
                                  avatarId: avatarId,
                                  enabled: enabled,
                                  modelId: sel.modelId,
                                  stockCount: sel.stockCount,
                                );
                              },
                            )
                          : isCart
                          ? GoToPaymentButton(
                              avatarId: avatarId,
                              enabled: avatarId.trim().isNotEmpty,
                            )
                          : isPayment
                          ? ConfirmPaymentButton(
                              avatarId: avatarId,
                              enabled: avatarId.trim().isNotEmpty,
                            )
                          : _FooterItem(
                              icon: Icons.qr_code_scanner,
                              onTap: () async {
                                final code = await showModalBottomSheet<String>(
                                  context: context,
                                  isScrollControlled: true,
                                  backgroundColor: Colors.black,
                                  builder: (_) => const _QrScanSheet(),
                                );

                                if (!context.mounted) return;
                                if (code == null || code.trim().isEmpty) return;

                                final target = _normalizeScannedToAppUri(
                                  code,
                                  currentFrom: from,
                                  avatarId: avatarId,
                                );

                                if (target == null) {
                                  ScaffoldMessenger.of(context).showSnackBar(
                                    const SnackBar(
                                      content: Text(
                                        'スキャン結果が無効です（Previewに遷移できません）',
                                      ),
                                    ),
                                  );
                                  return;
                                }

                                // ✅ 期待値: スキャンしたURLを以て preview へ遷移
                                context.go(target.toString());
                              },
                            ),
                    ),
                  ),

                  const SizedBox(width: 8),

                  // ✅ Avatar は /avatar へ（avatarId を引き継ぐ）
                  _AvatarIconButton(user: user, from: from, avatarId: avatarId),
                ],
              ),
            ),
          ),
        );
      },
    );
  }
}

class _FooterItem extends StatelessWidget {
  const _FooterItem({required this.icon, required this.onTap});

  final IconData icon;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return InkWell(
      borderRadius: BorderRadius.circular(12),
      onTap: onTap,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 6),
        child: Icon(icon, size: 22),
      ),
    );
  }
}

class _AvatarIconButton extends StatelessWidget {
  const _AvatarIconButton({
    required this.user,
    required this.from,
    required this.avatarId,
  });

  final User user;
  final String from;
  final String avatarId;

  @override
  Widget build(BuildContext context) {
    final photo = (user.photoURL ?? '').trim();
    final fallback = (user.displayName ?? user.email ?? user.uid).trim();
    final initial = fallback.isNotEmpty ? fallback[0].toUpperCase() : '?';

    return InkWell(
      borderRadius: BorderRadius.circular(999),
      onTap: () {
        final qp = <String, String>{
          'from': from,
          if (avatarId.trim().isNotEmpty) 'avatarId': avatarId.trim(),
        };
        final uri = Uri(path: '/avatar', queryParameters: qp);
        context.go(uri.toString());
      },
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 6),
        child: CircleAvatar(
          radius: 14,
          backgroundImage: photo.isNotEmpty ? NetworkImage(photo) : null,
          child: photo.isEmpty
              ? Text(initial, style: const TextStyle(fontSize: 12))
              : null,
        ),
      ),
    );
  }
}

/// Bottom sheet QR scanner
class _QrScanSheet extends StatefulWidget {
  const _QrScanSheet();

  @override
  State<_QrScanSheet> createState() => _QrScanSheetState();
}

class _QrScanSheetState extends State<_QrScanSheet> {
  final MobileScannerController _controller = MobileScannerController();
  bool _handled = false;

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  void _onDetect(BarcodeCapture capture) {
    if (_handled) return;
    final codes = capture.barcodes;
    if (codes.isEmpty) return;

    final raw = codes.first.rawValue;
    if (raw == null || raw.trim().isEmpty) return;

    _handled = true;
    Navigator.pop(context, raw.trim());
  }

  @override
  Widget build(BuildContext context) {
    final mq = MediaQuery.of(context);
    final h = mq.size.height;

    return SafeArea(
      top: false,
      child: SizedBox(
        height: h * 0.72,
        child: Stack(
          children: [
            MobileScanner(controller: _controller, onDetect: _onDetect),
            Positioned(
              left: 0,
              right: 0,
              top: 0,
              child: Padding(
                padding: const EdgeInsets.all(12),
                child: Row(
                  children: [
                    IconButton(
                      onPressed: () => Navigator.pop(context),
                      icon: const Icon(Icons.close, color: Colors.white),
                    ),
                    const SizedBox(width: 8),
                    const Text(
                      'Scan QR',
                      style: TextStyle(color: Colors.white, fontSize: 16),
                    ),
                    const Spacer(),
                    IconButton(
                      onPressed: () => _controller.toggleTorch(),
                      icon: const Icon(Icons.flash_on, color: Colors.white),
                    ),
                  ],
                ),
              ),
            ),
            Positioned(
              left: 0,
              right: 0,
              bottom: 18,
              child: Center(
                child: Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 12,
                    vertical: 8,
                  ),
                  decoration: BoxDecoration(
                    color: Colors.black.withValues(alpha: 0.55),
                    borderRadius: BorderRadius.circular(999),
                  ),
                  child: const Text(
                    'Point the camera at a QR code',
                    style: TextStyle(color: Colors.white),
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
