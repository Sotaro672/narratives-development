//frontend\sns\lib\app\shell\presentation\components\footer.dart
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:mobile_scanner/mobile_scanner.dart';

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
                          ? _AddToCartButton(
                              from: from,
                              listId: listId,
                              avatarId: avatarId,
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
/// いまは /cart に listId と from を渡すだけ（追加処理は cart 側で受ける想定）
/// ✅ 重要: avatarId を引き継ぐ（無いと cart が開けない）
class _AddToCartButton extends StatelessWidget {
  const _AddToCartButton({
    required this.from,
    required this.listId,
    required this.avatarId,
  });

  final String from;
  final String listId;
  final String avatarId;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: 40,
      child: ElevatedButton.icon(
        icon: const Icon(Icons.add_shopping_cart_outlined, size: 20),
        label: const Text('カートに入れる'),
        onPressed: () {
          final qp = <String, String>{
            'from': from,
            if (listId.trim().isNotEmpty) 'listId': listId.trim(),
            // cart 側で「追加イベント」として扱いたい場合に使う
            'action': 'addFromCatalog',

            // ✅ cart は avatarId 必須
            if (avatarId.trim().isNotEmpty) 'avatarId': avatarId.trim(),
          };

          final uri = Uri(path: '/cart', queryParameters: qp);
          context.go(uri.toString());
        },
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
