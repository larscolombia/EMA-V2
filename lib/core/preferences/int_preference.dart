import 'package:shared_preferences/shared_preferences.dart';

class IntPreference {
  // late SharedPreferences? _prefs;
  final String key;
  final int defaultValue;
  int? _value;

  IntPreference({
    required this.key,
    required this.defaultValue
  });

  Future<int> getValue() async {
    final prefs = await SharedPreferences.getInstance();
    _value = prefs.getInt(key) ?? defaultValue;
    return _value!;
  }

  setValue(int value) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setInt(key, value);
    _value = value;
  }
}
