// frontend\mall\lib\features\auth\presentation\hook\use_billing_address.dart
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../application/billing_address_service.dart';

/// BillingAddressPage の状態/処理（ChangeNotifier で hook 風）
class UseBillingAddress extends ChangeNotifier {
  UseBillingAddress({required this.from})
    : _service = BillingAddressService(
        logger: (s) {
          if (!kDebugMode) return;
          debugPrint('[UseBillingAddress] $s');
        },
      );

  /// optional back route
  final String? from;

  final BillingAddressService _service;

  final cardNumberCtrl = TextEditingController();
  final cardHolderCtrl = TextEditingController();
  final cvcCtrl = TextEditingController();

  bool saving = false;
  String? msg;

  @override
  void dispose() {
    cardNumberCtrl.dispose();
    cardHolderCtrl.dispose();
    cvcCtrl.dispose();
    _service.dispose();
    super.dispose();
  }

  String backTo() => _service.backTo(from);

  bool get canSave {
    return _service.canSave(
      saving: saving,
      cardNumberRaw: cardNumberCtrl.text,
      holderRaw: cardHolderCtrl.text,
      cvcRaw: cvcCtrl.text,
    );
  }

  void onCardNumberChanged() {
    final current = cardNumberCtrl.text;
    final formatted = _service.formatCardNumberForDisplay(current);
    if (formatted == current) return;

    final sel = cardNumberCtrl.selection;
    cardNumberCtrl.value = TextEditingValue(
      text: formatted,
      selection: TextSelection.collapsed(
        offset: (sel.baseOffset + (formatted.length - current.length)).clamp(
          0,
          formatted.length,
        ),
      ),
    );

    notifyListeners();
  }

  void onFormChanged() {
    notifyListeners();
  }

  /// ✅ UI側の責務: 状態制御・メッセージ表示・画面遷移
  Future<void> save(BuildContext context) async {
    saving = true;
    msg = null;
    notifyListeners();

    try {
      final res = await _service.save(
        cardNumberRaw: cardNumberCtrl.text,
        holderRaw: cardHolderCtrl.text,
        cvcRaw: cvcCtrl.text,
      );

      msg = res.message;

      if (res.ok && res.nextRoute != null) {
        if (!context.mounted) return;
        context.go(res.nextRoute!);
      }
    } finally {
      saving = false;
      notifyListeners();
    }
  }
}
