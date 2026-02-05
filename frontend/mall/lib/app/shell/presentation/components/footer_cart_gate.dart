import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import 'package:mall/features/cart/presentation/hook/use_cart.dart';

import '../../../routing/routes.dart';
import 'footer_buttons.dart';

/// /cart のときだけ UseCartController を生存させる責務
class FooterCartGate {
  FooterCartGate._();

  static void syncCartControllerIfNeeded({
    required bool isCart,
    required BuildContext context,
    required UseCartController? current,
    required void Function(UseCartController? next) setController,
  }) {
    if (isCart && current == null) {
      final c = UseCartController(context: context);
      c.init();
      setController(c);
      return;
    }
    if (!isCart && current != null) {
      current.dispose();
      setController(null);
      return;
    }
  }
}

/// ✅ /cart 専用：UseCartController を参照して「購入する」押下可否を決める
class CartAwareGoToPaymentButton extends StatelessWidget {
  const CartAwareGoToPaymentButton({
    super.key,
    required this.avatarId,
    required this.controller,
  });

  final String avatarId;
  final UseCartController? controller;

  @override
  Widget build(BuildContext context) {
    final aid = avatarId.trim();

    // controller が無い（= /cart 以外）ケースは防御的に disabled
    final uc = controller;
    if (uc == null) {
      return GoToPaymentButton(avatarId: aid, enabled: false);
    }

    return FutureBuilder<CartDTO>(
      future: uc.future,
      builder: (context, snap) {
        final _ = snap.data; // ignore: unused_local_variable

        final res = uc.buildResult((fn) {});
        final cartNotEmpty = !res.isEmpty;

        final enabled = aid.isNotEmpty && cartNotEmpty;

        return GoToPaymentButton(avatarId: aid, enabled: enabled);
      },
    );
  }
}

/// Signed-in footer が使う「現在地保持」helper
String currentLocationForReturnTo(BuildContext context) {
  return GoRouterState.of(context).uri.toString();
}

/// Signed-in footer が使う「path判定」helper
bool isCartPath(BuildContext context) {
  final path = GoRouterState.of(context).uri.path;
  return path == AppRoutePath.cart;
}
