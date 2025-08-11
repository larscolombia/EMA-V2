enum ClinicalCaseType {
  /// Equivalente a static en Laravel (no se usa static aquí porque es palabra reservada)
  analytical,
  interactive;

  factory ClinicalCaseType.fromName(String name) {
    return name == 'static'
      ? ClinicalCaseType.analytical
      : ClinicalCaseType.interactive;
  }

  String get name {
    return this == ClinicalCaseType.analytical
      ? 'static'
      : 'interactive'; // Todo: revisar valor asignado en Laravel > en el controlador.
  }

  String get description {
    return this == ClinicalCaseType.analytical
      ? 'Caso Clínico Analítico'
      : 'Caso Clínico Interactivo';
  }
}
