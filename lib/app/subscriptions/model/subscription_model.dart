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
    return Subscription(
      id: json['id'],
      name: plan != null ? plan['name'] : json['name'],
      currency: plan != null ? plan['currency'] : json['currency'],
      price: plan != null
          ? double.parse(plan['price'].toString())
          : json['price'].toDouble(),
      billing: plan != null ? 'Mensual' : json['billing'],
      consultations:
          json['consultations'] ?? (plan != null ? plan['consultations'] : 0),
      questionnaires:
          json['questionnaires'] ?? (plan != null ? plan['questionnaires'] : 0),
      clinicalCases:
          json['clinical_cases'] ?? (plan != null ? plan['clinical_cases'] : 0),
      files: json['files'] ?? (plan != null ? plan['files'] : 0),
      frequency: json['frequency'] ?? (plan != null ? plan['frequency'] : 0),
      endDate:
          json['end_date'] != null ? DateTime.parse(json['end_date']) : null,
      statistics: json['statistics'] ?? (plan != null ? plan['statistics'] : 0),
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
