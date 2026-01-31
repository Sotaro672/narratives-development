// frontend/inspector/lib/models/inspector_color.dart
class InspectorColor {
  final int rgb; // ARGB or RGB int
  final String? name;

  InspectorColor({required this.rgb, this.name});

  factory InspectorColor.fromJson(Map<String, dynamic> json) {
    final raw = json['rgb'];
    int rgbValue;

    if (raw is int) {
      rgbValue = raw;
    } else if (raw is String) {
      rgbValue = int.tryParse(raw) ?? 0xff000000;
    } else {
      rgbValue = 0xff000000;
    }

    return InspectorColor(rgb: rgbValue, name: json['name'] as String?);
  }
}
