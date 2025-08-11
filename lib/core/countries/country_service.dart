import 'dart:convert';
import 'package:ema_educacion_medica_avanzada/config/constants/constants.dart';
import 'package:ema_educacion_medica_avanzada/core/countries/country_model.dart';
import 'package:http/http.dart' as http;

class CountryService {
  List<CountryModel>? _cachedCountries;

  Future<List<CountryModel>> getCountries() async {
    if (_cachedCountries != null) {
      return _cachedCountries!;
    }

    try {
      final url = Uri.parse('$apiUrl/countries');
      final response = await http.get(url);

      if (response.statusCode == 200) {
        final List<dynamic> data = json.decode(response.body);
        _cachedCountries =
            data.map((country) => CountryModel.fromJson(country)).toList();
        return _cachedCountries!;
      } else {
        throw Exception(
            'Error al obtener la lista de países: ${response.statusCode}');
      }
    } catch (e) {
      throw Exception('Error al obtener la lista de países: $e');
    }
  }
}
