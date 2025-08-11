class MonthlyScore {
  final int mes;
  final int puntos;

  MonthlyScore({
    required this.mes,
    required this.puntos,
  });

  factory MonthlyScore.fromJson(Map<String, dynamic> json) {
    return MonthlyScore(
      mes: json['mes'] as int,
      puntos: json['puntos'] as int,
    );
  }

  // Nuevo método para convertir a mapa
  Map<String, dynamic> toJson() => {
        'mes': mes,
        'puntos': puntos,
      };

  // Sobrescribe toString para incluir la información real
  @override
  String toString() => toJson().toString();
}
