class Subscription {
  final int id;
  final String name;
  final String currency;
  final double price;
  final String billing;
  final int consultations;
  final int questionnaires;
  final int clinicalCases;
  final int files;
  final int? frequency;
  final DateTime? endDate;
  final int statistics;

  Subscription({
    required this.id,
    required this.name,
    required this.currency,
    required this.price,
    required this.billing,
    required this.consultations,
    required this.questionnaires,
    required this.clinicalCases,
    required this.files,
    this.frequency,
    this.endDate,
    this.statistics = 0, // valor por defecto
  });

  factory Subscription.fromJson(Map<String, dynamic> json) {
    final plan = json['subscription_plan'];

    int _asInt(dynamic v, {int def = 0}) {
      if (v == null) return def;
      if (v is int) return v;
      if (v is num) return v.toInt();
      final s = v.toString();
      return int.tryParse(s) ?? def;
    }

    double _asDouble(dynamic v, {double def = 0.0}) {
      if (v == null) return def;
      if (v is num) return v.toDouble();
      return double.tryParse(v.toString()) ?? def;
    }

    String _asString(dynamic v, {String def = ''}) {
      if (v == null) return def;
      return v.toString();
    }

    final name =
        plan != null ? _asString(plan['name']) : _asString(json['name']);
    final currency =
        plan != null
            ? _asString(plan['currency'])
            : _asString(json['currency']);
    final price =
        plan != null ? _asDouble(plan['price']) : _asDouble(json['price']);
    final billing = plan != null ? 'Mensual' : _asString(json['billing']);

    return Subscription(
      id: _asInt(json['id']), // Defaults to 0 if missing/null
      name: name,
      currency: currency,
      price: price,
      billing: billing,
      consultations: _asInt(
        json['consultations'],
        def: plan != null ? _asInt(plan['consultations']) : 0,
      ),
      questionnaires: _asInt(
        json['questionnaires'],
        def: plan != null ? _asInt(plan['questionnaires']) : 0,
      ),
      clinicalCases: _asInt(
        json['clinical_cases'],
        def: plan != null ? _asInt(plan['clinical_cases']) : 0,
      ),
      files: _asInt(
        json['files'],
        def: plan != null ? _asInt(plan['files']) : 0,
      ),
      frequency:
          json.containsKey('frequency')
              ? _asInt(json['frequency'])
              : (plan != null ? _asInt(plan['frequency']) : 0),
      endDate:
          json['end_date'] != null && json['end_date'].toString().isNotEmpty
              ? DateTime.tryParse(json['end_date'].toString())
              : null,
      statistics: _asInt(
        json['statistics'],
        def: plan != null ? _asInt(plan['statistics']) : 0,
      ),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'name': name,
      'currency': currency,
      'price': price,
      'billing': billing,
      'consultations': consultations,
      'questionnaires': questionnaires,
      'clinical_cases': clinicalCases,
      'files': files,
      'frequency': frequency,
      'end_date': endDate?.toIso8601String(),
      'statistics': statistics,
    };
  }

  Subscription copyWith({
    int? id,
    String? name,
    String? currency,
    double? price,
    String? billing,
    int? consultations,
    int? questionnaires,
    int? clinicalCases,
    int? files,
    int? frequency,
    DateTime? endDate,
    int? statistics,
  }) {
    return Subscription(
      id: id ?? this.id,
      name: name ?? this.name,
      currency: currency ?? this.currency,
      price: price ?? this.price,
      billing: billing ?? this.billing,
      consultations: consultations ?? this.consultations,
      questionnaires: questionnaires ?? this.questionnaires,
      clinicalCases: clinicalCases ?? this.clinicalCases,
      files: files ?? this.files,
      frequency: frequency ?? this.frequency,
      endDate: endDate ?? this.endDate,
      statistics: statistics ?? this.statistics,
    );
  }
}
