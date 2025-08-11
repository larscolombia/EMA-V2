class TestScore {
  final int testId;
  final String testName;
  final int scoreObtained;
  final int maxScore;

  TestScore({
    required this.testId,
    required this.testName,
    required this.scoreObtained,
    required this.maxScore,
  });

  factory TestScore.fromJson(Map<String, dynamic> json) {
    return TestScore(
      testId: json['test_id'],
      testName: json['test_name'],
      scoreObtained: json['score_obtained'],
      maxScore: json['max_score'],
    );
  }
}

class TestProgressSummary {
  final int totalTests;
  final int totalPreguntas;
  final int totalCorrectas;
  final int totalIncorrectas;

  TestProgressSummary({
    required this.totalTests,
    required this.totalPreguntas,
    required this.totalCorrectas,
    required this.totalIncorrectas,
  });

  factory TestProgressSummary.fromJson(Map<String, dynamic> json) {
    return TestProgressSummary(
      totalTests: json['total_tests'],
      totalPreguntas: json['total_preguntas'],
      totalCorrectas: json['total_correctas'],
      totalIncorrectas: json['total_incorrectas'],
    );
  }
}

class TestProgressData {
  final List<TestScore> tests;
  final TestProgressSummary summary;

  TestProgressData({
    required this.tests,
    required this.summary,
  });

  factory TestProgressData.fromJson(Map<String, dynamic> json) {
    final data = json['data'];
    final tests = data['tests'] != null
        ? (data['tests'] as List)
            .map((item) => TestScore.fromJson(item as Map<String, dynamic>))
            .toList()
        : <TestScore>[];
    final summary = TestProgressSummary.fromJson(data['resumen']);
    return TestProgressData(tests: tests, summary: summary);
  }
}
