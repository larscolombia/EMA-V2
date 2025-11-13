class MonthlyScore {
  final String mes;         // Formato: "2024-11" (año-mes)
  final int puntos;         // Total de puntos obtenidos en el mes
  final int testsCount;     // Cantidad de tests realizados en el mes

  MonthlyScore({
    required this.mes,
    required this.puntos,
    required this.testsCount,
  });

  factory MonthlyScore.fromJson(Map<String, dynamic> json) {
    return MonthlyScore(
      mes: json['mes'] as String,
      puntos: json['puntos'] as int,
      testsCount: json['tests_count'] as int? ?? 0,
    );
  }

  // Nuevo método para convertir a mapa
  Map<String, dynamic> toJson() => {
        'mes': mes,
        'puntos': puntos,
        'tests_count': testsCount,
      };

  // Sobrescribe toString para incluir la información real
  @override
  String toString() => toJson().toString();
}
