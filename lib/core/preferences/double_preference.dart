import 'package:shared_preferences/shared_preferences.dart';

class DoublePreference {
  // late SharedPreferences? _prefs;
  final String key;
  final double defaultValue;
  double? _value;

  DoublePreference({
    required this.key,
    required this.defaultValue
  });

  Future<double> getValue() async {
    final prefs = await SharedPreferences.getInstance();
    _value = prefs.getDouble(key)?? defaultValue;
    return _value!;
  }

  setValue(double value) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setDouble(key, value);
    _value = value;
  }
}
