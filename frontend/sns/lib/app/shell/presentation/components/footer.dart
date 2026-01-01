// frontend\sns\lib\app\shell\presentation\components\footer.dart
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:mobile_scanner/mobile_scanner.dart';

import 'package:sns/features/cart/infrastructure/cart_repository_http.dart';

import '../state/catalog_selection_store.dart';

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
    return (qp['avatarId'] ?? '').trim();
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
    if (!merged.containsKey('avatarId')) {
      final aid = (current.queryParameters['avatarId'] ?? '').trim();
      if (aid.isNotEmpty) merged['avatarId'] = aid;
    }

    return Uri(path: path, queryParameters: merged.isEmpty ? null : merged);
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
                    label: 'Shop',
                    onTap: () {
                      final uri = _buildUriPreserveAvatarId(context, '/');
                      context.go(uri.toString());
                    },
                  ),
                  const SizedBox(width: 8),

                  // ✅ 中央：catalog では「カートに入れる」ボタン、それ以外は Scan
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

                                // ✅ NEW: backend が必須にした inventoryId/listId を揃える
                                final invId = sel.inventoryId.trim();

                                // ✅ 在庫 0 は押下不可 + 必須IDが揃っていること
                                final enabled =
                                    sameList &&
                                    invId.isNotEmpty &&
                                    listId.trim().isNotEmpty &&
                                    mid.isNotEmpty &&
                                    stock > 0;

                                return _AddToCartButton(
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
                          : _FooterItem(
                              icon: Icons.qr_code_scanner,
                              label: 'Scan',
                              onTap: () async {
                                final code = await showModalBottomSheet<String>(
                                  context: context,
                                  isScrollControlled: true,
                                  backgroundColor: Colors.black,
                                  builder: (_) => const _QrScanSheet(),
                                );

                                if (!context.mounted) return;
                                if (code == null || code.trim().isEmpty) return;

                                await showDialog<void>(
                                  context: context,
                                  builder: (_) => AlertDialog(
                                    title: const Text('Scanned'),
                                    content: Text(code),
                                    actions: [
                                      TextButton(
                                        onPressed: () => Navigator.pop(context),
                                        child: const Text('OK'),
                                      ),
                                    ],
                                  ),
                                );
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
  const _FooterItem({
    required this.icon,
    required this.label,
    required this.onTap,
  });

  final IconData icon;
  final String label;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final t = Theme.of(context).textTheme;

    return InkWell(
      borderRadius: BorderRadius.circular(12),
      onTap: onTap,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 6),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, size: 22),
            const SizedBox(height: 4),
            Text(label, style: t.labelSmall),
          ],
        ),
      ),
    );
  }
}

/// ✅ catalog 用：カートに入れる CTA
/// ✅ 「model が 1つに絞れたら enabled」
/// ✅ 在庫 0 は押下不可
/// ✅ 押下時に CartHandler に add リクエストを投げてから /cart へ遷移する
class _AddToCartButton extends StatefulWidget {
  const _AddToCartButton({
    required this.from,
    required this.inventoryId,
    required this.listId,
    required this.avatarId,
    required this.enabled,
    required this.modelId,
    required this.stockCount,
  });

  final String from;

  // ✅ NEW: required by backend
  final String inventoryId;
  final String listId;

  final String avatarId;

  final bool enabled;
  final String? modelId;
  final int? stockCount;

  @override
  State<_AddToCartButton> createState() => _AddToCartButtonState();
}

class _AddToCartButtonState extends State<_AddToCartButton> {
  bool _loading = false;

  Future<void> _addThenGoCart() async {
    final mid = (widget.modelId ?? '').trim();
    final sc = widget.stockCount ?? 0;
    final aid = widget.avatarId.trim();
    final invId = widget.inventoryId.trim();
    final listId = widget.listId.trim();

    // ✅ 最終ガード（backend 必須フィールドも含める）
    final canTap =
        widget.enabled &&
        !_loading &&
        aid.isNotEmpty &&
        invId.isNotEmpty &&
        listId.isNotEmpty &&
        mid.isNotEmpty &&
        sc > 0;
    if (!canTap) return;

    setState(() => _loading = true);

    try {
      final repo = CartRepositoryHttp();
      try {
        await repo.addItem(
          avatarId: aid,
          inventoryId: invId,
          listId: listId,
          modelId: mid,
          qty: 1,
        );
      } finally {
        repo.dispose();
      }

      if (!mounted) return;

      // ✅ 追加できたら cart へ
      final qp = <String, String>{'from': widget.from, 'avatarId': aid};
      final uri = Uri(path: '/cart', queryParameters: qp);

      ScaffoldMessenger.of(
        context,
      ).showSnackBar(const SnackBar(content: Text('カートに追加しました')));

      context.go(uri.toString());
    } catch (e) {
      if (!mounted) return;
      final msg = e.toString();
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('追加に失敗しました: $msg')));
    } finally {
      if (mounted) {
        setState(() => _loading = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final mid = (widget.modelId ?? '').trim();
    final sc = widget.stockCount ?? 0;

    final invId = widget.inventoryId.trim();
    final listId = widget.listId.trim();

    // ✅ 最終ガード：enabled が true でも在庫 0 は絶対押下不可
    final canTap =
        widget.enabled &&
        !_loading &&
        invId.isNotEmpty &&
        listId.isNotEmpty &&
        mid.isNotEmpty &&
        sc > 0;

    final label = _loading
        ? '追加中...'
        : (mid.isNotEmpty && sc <= 0)
        ? '在庫なし'
        : 'カートに入れる';

    return SizedBox(
      height: 40,
      child: ElevatedButton.icon(
        icon: _loading
            ? const SizedBox(
                width: 18,
                height: 18,
                child: CircularProgressIndicator(strokeWidth: 2),
              )
            : const Icon(Icons.add_shopping_cart_outlined, size: 20),
        label: Text(label),
        onPressed: (!canTap) ? null : _addThenGoCart,
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
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            CircleAvatar(
              radius: 14,
              backgroundImage: photo.isNotEmpty ? NetworkImage(photo) : null,
              child: photo.isEmpty
                  ? Text(initial, style: const TextStyle(fontSize: 12))
                  : null,
            ),
            const SizedBox(height: 4),
            Text('Avatar', style: Theme.of(context).textTheme.labelSmall),
          ],
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
