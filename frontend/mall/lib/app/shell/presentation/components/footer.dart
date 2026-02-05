import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:mobile_scanner/mobile_scanner.dart';

import '../state/catalog_selection_store.dart';

// ✅ routing constants
import '../../../routing/routes.dart';

// ✅ Pattern B stores (no URL-based navigation state)
import '../../../routing/navigation.dart';

// ✅ 既存：3 buttons extracted
import 'footer_buttons.dart';

// ✅ Cart (Pattern A stats): use_cart controller/result
import 'package:mall/features/cart/presentation/hook/use_cart.dart';

// ✅ MeAvatar model (now carries avatar patch fields)
import 'package:mall/features/avatar/presentation/model/me_avatar.dart';

// ✅ NEW split parts
import 'footer_qr_nav.dart';
import 'footer_avatar_future.dart';
import 'footer_cart_gate.dart';

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
///
/// Pattern B:
/// - `from` を URL query に入れない
/// - `avatarId` を URL query に入れない／URL から読まない
/// - 戻り先は NavStore に保持する
/// - avatarId は AvatarIdStore（= app state）を参照する
///
/// ✅ 方針B：/cart のときだけ UseCartController を参照して「購入する」押下可否を決める
class SignedInFooter extends StatefulWidget {
  const SignedInFooter({super.key});

  @override
  State<SignedInFooter> createState() => _SignedInFooterState();
}

class _SignedInFooterState extends State<SignedInFooter> {
  UseCartController? _cartUc;

  bool _isCatalogPath(BuildContext context) {
    final path = GoRouterState.of(context).uri.path;
    return path.startsWith('/catalog/');
  }

  bool _isPaymentPath(BuildContext context) {
    final path = GoRouterState.of(context).uri.path;
    return path == AppRoutePath.payment;
  }

  String _catalogListIdOrEmpty(BuildContext context) {
    final path = GoRouterState.of(context).uri.path; // /catalog/:listId
    final parts = path.split('/');
    if (parts.length >= 3 && parts[1] == 'catalog') return parts[2];
    return '';
  }

  @override
  void dispose() {
    _cartUc?.dispose();
    _cartUc = null;
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return StreamBuilder<User?>(
      // ✅ auth state だけを見る
      stream: FirebaseAuth.instance.userChanges(),
      builder: (context, snap) {
        final user = FirebaseAuth.instance.currentUser ?? snap.data;
        if (user == null) return const SizedBox.shrink();

        final isCatalog = _isCatalogPath(context);
        final isCart = isCartPath(context);
        final isPayment = _isPaymentPath(context);

        // ✅ /cart のときだけ cart controller を生かす
        FooterCartGate.syncCartControllerIfNeeded(
          isCart: isCart,
          context: context,
          current: _cartUc,
          setController: (next) {
            setState(() {
              _cartUc = next;
            });
          },
        );

        final listId = isCatalog ? _catalogListIdOrEmpty(context) : '';

        // ✅ Pattern B: avatarId は URL から読まない（store を参照）
        final avatarId = AvatarIdStore.I.avatarId.trim();

        // ✅ Pattern B: 戻り先は store に保持（URLへは出さない）
        final returnTo = currentLocationForReturnTo(context);

        return AvatarProfileFuture(
          builder: (context, MeAvatar? profile) {
            final avatarIcon = profile?.avatarIcon;

            return Material(
              color: Theme.of(context).cardColor,
              elevation: 0,
              child: SafeArea(
                top: false,
                child: Padding(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 18,
                    vertical: 10,
                  ),
                  child: Row(
                    children: [
                      // ✅ Shop (Home)
                      _FooterItem(
                        icon: Icons.storefront_outlined,
                        onTap: () {
                          NavStore.I.setReturnTo(returnTo);
                          context.go(AppRoutePath.home);
                        },
                      ),
                      const SizedBox(width: 8),

                      // ✅ 中央：
                      // - catalog: カートに入れる
                      // - cart: 購入する（paymentへ）※ cart=0 のときは押下不可
                      // - payment: 支払を確定する
                      // - other: Scan（スキャンしたURLへ遷移）
                      Expanded(
                        child: Center(
                          child: isCatalog
                              ? ValueListenableBuilder<CatalogSelection>(
                                  valueListenable:
                                      CatalogSelectionStore.notifier,
                                  builder: (context, sel, _) {
                                    final sameList =
                                        sel.listId.trim() == listId.trim();

                                    final mid = (sel.modelId ?? '').trim();
                                    final stock = sel.stockCount ?? 0;

                                    // ✅ backend が必須にした inventoryId/listId を揃える
                                    final invId = sel.inventoryId.trim();

                                    final enabled =
                                        sameList &&
                                        invId.isNotEmpty &&
                                        listId.trim().isNotEmpty &&
                                        mid.isNotEmpty &&
                                        stock > 0;

                                    return AddToCartButton(
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
                              ? CartAwareGoToPaymentButton(
                                  avatarId: avatarId,
                                  controller: _cartUc,
                                )
                              : isPayment
                              ? ConfirmPaymentButton(
                                  avatarId: avatarId,
                                  enabled: avatarId.isNotEmpty,
                                )
                              : _FooterItem(
                                  icon: Icons.qr_code_scanner,
                                  onTap: () async {
                                    final code =
                                        await showModalBottomSheet<String>(
                                          context: context,
                                          isScrollControlled: true,
                                          backgroundColor: Colors.black,
                                          builder: (_) => const _QrScanSheet(),
                                        );

                                    if (!context.mounted) return;
                                    if (code == null || code.trim().isEmpty) {
                                      return;
                                    }

                                    final target =
                                        FooterQrNav.normalizeScannedToAppUri(
                                          code,
                                        );
                                    if (target == null) {
                                      showInvalidScanSnackBar(context);
                                      return;
                                    }

                                    NavStore.I.setReturnTo(returnTo);
                                    context.go(target.toString());
                                  },
                                ),
                        ),
                      ),

                      const SizedBox(width: 8),

                      // ✅ Avatar: /avatar へ（query なし）
                      _AvatarIconButton(
                        avatarIcon: avatarIcon,
                        fallbackText:
                            (user.displayName ?? user.email ?? user.uid).trim(),
                        onTap: () {
                          NavStore.I.setReturnTo(returnTo);
                          context.go(AppRoutePath.avatar);
                        },
                      ),
                    ],
                  ),
                ),
              ),
            );
          },
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
    required this.avatarIcon,
    required this.fallbackText,
    required this.onTap,
  });

  /// ✅ absolute schema: avatarIcon
  final String? avatarIcon;

  final String fallbackText;
  final VoidCallback onTap;

  bool _isHttpUrl(String? v) {
    final s = (v ?? '').trim();
    if (s.isEmpty) return false;
    return s.startsWith('http://') || s.startsWith('https://');
  }

  @override
  Widget build(BuildContext context) {
    final icon = (avatarIcon ?? '').trim();
    final fallback = fallbackText.trim();
    final initial = fallback.isNotEmpty ? fallback[0].toUpperCase() : '?';

    return InkWell(
      borderRadius: BorderRadius.circular(999),
      onTap: onTap,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 6),
        child: CircleAvatar(
          radius: 14,
          // ✅ NetworkImage は http(s) のみ（gs:// や相対pathは表示不可なので fallback）
          backgroundImage: _isHttpUrl(icon) ? NetworkImage(icon) : null,
          child: !_isHttpUrl(icon)
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
