class Plan {
  final int? id;
  final String name;
  final String currency;
  final double price;
  final String billing;
  final int consultations;
  final int questionnaires;
  final int clinicalCases;
  final int files;
  final String? stripeProductId;
  final String? stripePriceId;
  final int statistics;

  Plan({
    this.id,
    required this.name,
    required this.currency,
    required this.price,
    required this.billing,
    required this.consultations,
    required this.questionnaires,
    required this.clinicalCases,
    required this.files,
    this.stripeProductId,
    this.stripePriceId,
    this.statistics = 0,
  });

  factory Plan.fromJson(Map<String, dynamic> json) {
    return Plan(
      id: json['id'] as int?,
      name: json['name'] as String,
      currency: json['currency'] as String,
      price: (json['price'] as num).toDouble(),
      billing: json['billing'] as String,
      consultations: json['consultations'] as int,
      questionnaires: json['questionnaires'] as int,
      clinicalCases: json['clinical_cases'] as int,
      files: json['files'] as int,
      stripeProductId: json['stripe_product_id'] as String?,
      stripePriceId: json['stripe_price_id'] as String?,
      statistics: json['statistics'] as int? ?? 0,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      if (id != null) 'id': id,
      'name': name,
      'currency': currency,
      'price': price,
      'billing': billing,
      'consultations': consultations,
      'questionnaires': questionnaires,
      'clinical_cases': clinicalCases,
      'files': files,
      if (stripeProductId != null) 'stripe_product_id': stripeProductId,
      if (stripePriceId != null) 'stripe_price_id': stripePriceId,
      'statistics': statistics,
    };
  }

  Plan copyWith({
    int? id,
    String? name,
    String? currency,
    double? price,
    String? billing,
    int? consultations,
    int? questionnaires,
    int? clinicalCases,
    int? files,
    String? stripeProductId,
    String? stripePriceId,
    int? statistics,
  }) {
    return Plan(
      id: id ?? this.id,
      name: name ?? this.name,
      currency: currency ?? this.currency,
      price: price ?? this.price,
      billing: billing ?? this.billing,
      consultations: consultations ?? this.consultations,
      questionnaires: questionnaires ?? this.questionnaires,
      clinicalCases: clinicalCases ?? this.clinicalCases,
      files: files ?? this.files,
      stripeProductId: stripeProductId ?? this.stripeProductId,
      stripePriceId: stripePriceId ?? this.stripePriceId,
      statistics: statistics ?? this.statistics,
    );
  }

  String get billingLabel => billing == 'Mensual' ? 'Mensual' : 'Anual';

  String get formattedPrice {
    final symbol = currency == 'USD' ? '\$' : currency;
    return '$symbol${price.toStringAsFixed(2)}';
  }
}
