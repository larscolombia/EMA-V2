// ignore_for_file: public_member_api_docs, sort_constructors_first
import 'dart:convert';

import 'package:ema_educacion_medica_avanzada/app/clinical_cases/model/clinical_case_type.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/model/life_stage.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/model/sex_and_status.dart';
import 'package:uuid/uuid.dart';


class ClinicalCaseModel {
  // ids
  final String uid;
  final int remoteId;
  final String threadId;
  final int userId;
  // general
  final String title;
  final ClinicalCaseType type;
  // patient profile
  final String age;
  final String sex;
  final bool gestante;
  final bool isReal;
  // contents
  final String anamnesis;
  final String physicalExamination;
  final String diagnosticTests;
  final String finalDiagnosis;
  final String management;
  // Datetimes
  final DateTime createdAt;
  final DateTime updatedAt;

  final String? feedback;
  final String? summary; // Resumen corto del caso (máx. 2 líneas)

  ClinicalCaseModel({
    required this.uid,
    required this.remoteId,
    required this.threadId,
    required this.userId,
    required this.title,
    required this.type,
    required this.age,
    required this.sex,
    required this.gestante,
    required this.isReal,
    required this.anamnesis,
    required this.physicalExamination,
    required this.diagnosticTests,
    required this.finalDiagnosis,
    required this.management,
    required this.createdAt,
    required this.updatedAt,
    this.feedback,
    this.summary,
  });

  ClinicalCaseModel copyWith({
    String? uid,
    int? remoteId,
    String? threadId,
    int? userId,
    String? title,
    ClinicalCaseType? type,
    String? age,
    String? sex,
    bool? gestante,
    bool? isReal,
    String? anamnesis,
    String? physicalExamination,
    String? diagnosticTests,
    String? finalDiagnosis,
    String? management,
    DateTime? createdAt,
    DateTime? updatedAt,
    String? feedback,
    String? summary,
  }) {
    return ClinicalCaseModel(
      uid: uid ?? this.uid,
      remoteId: remoteId ?? this.remoteId,
      threadId: threadId ?? this.threadId,
      userId: userId ?? this.userId,
      title: title ?? this.title,
      type: type ?? this.type,
      age: age ?? this.age,
      sex: sex ?? this.sex,
      gestante: gestante ?? this.gestante,
      isReal: isReal ?? this.isReal,
      anamnesis: anamnesis ?? this.anamnesis,
      physicalExamination: physicalExamination ?? this.physicalExamination,
      diagnosticTests: diagnosticTests ?? this.diagnosticTests,
      finalDiagnosis: finalDiagnosis ?? this.finalDiagnosis,
      management: management ?? this.management,
      createdAt: createdAt ?? this.createdAt,
      updatedAt: updatedAt ?? this.updatedAt,
      feedback: feedback ?? this.feedback,
      summary: summary ?? this.summary,
    );
  }

  factory ClinicalCaseModel.generate({
    required int userId,
    required ClinicalCaseType type,
    required LifeStage lifeStage,
    required SexAndStatus sexAndStatus
  }) {
    final String age = lifeStage.age;
    final String sex = sexAndStatus.sex;
    final bool gestante = sexAndStatus.isPregnant;

    return ClinicalCaseModel(
      uid: Uuid().v4(),
      remoteId: 0,
      threadId: '',
      userId: userId,
      title: '',
      type: type,
      age: age,
      sex: sex,
      gestante: gestante,
      isReal: true,
      anamnesis: '',
      physicalExamination: '',
      diagnosticTests: '',
      finalDiagnosis: '',
      management: '',
      createdAt: DateTime.now(),
      updatedAt: DateTime.now()
    );
  }
  
  factory ClinicalCaseModel.fromApi(Map<String, dynamic> map) {
    final Map<String, dynamic> data =
        (map['case'] ?? map['data']) as Map<String, dynamic>;

    return ClinicalCaseModel(
      uid: '', // Se asignará externamente
      remoteId: data['id'] is int
          ? data['id'] as int
          : int.tryParse(data['id'].toString()) ?? 0,
      threadId: map['thread_id'] as String? ?? '',
      userId: 0,
      title: data['title'] != null ? data['title'] as String : '',
      type: ClinicalCaseType.fromName(
          (data['type'] ?? data['case_type'] ?? 'static') as String),
      age: data['age'] as String? ?? '',
      sex: data['sex'] as String? ?? '',
      gestante: data['gestante'] == 1 || data['pregnant'] == 1,
      isReal: data['is_real'] == 1 || true,
      anamnesis: data['anamnesis'] as String? ?? '',
      physicalExamination: data['physical_examination'] as String? ?? '',
      diagnosticTests: data['diagnostic_tests'] as String? ?? '',
      finalDiagnosis: data['final_diagnosis'] as String? ?? '',
      management: data['management'] as String? ?? '',
      createdAt: data['created_at'] != null
          ? DateTime.parse(data['created_at'] as String)
          : DateTime.now(),
      updatedAt: data['updated_at'] != null
          ? DateTime.parse(data['updated_at'] as String)
          : DateTime.now(),
      feedback: data['feedback'] != null ? map['feedback'] as String : null,
      summary: null, // Los casos remotos no tienen summary local
    );
  }

  factory ClinicalCaseModel.fromMap(Map<String, dynamic> map) {
    return ClinicalCaseModel(
      uid: map['uid'] as String,
      remoteId: map['remoteId'] as int,
      threadId: map['threadId'] as String,
      userId: map['userId'] as int,
      title: map['title'] as String,
      type: ClinicalCaseType.fromName(map['type'] as String),
      age: map['age'] as String,
      sex: map['sex'] as String,
      gestante: map['gestante'] == 1,
      isReal: map['isReal'] == 1,
      anamnesis: map['anamnesis'] as String,
      physicalExamination: map['physicalExamination'] as String,
      diagnosticTests: map['diagnosticTests'] as String,
      finalDiagnosis: map['finalDiagnosis'] as String,
      management: map['management'] as String,
      createdAt: DateTime.fromMillisecondsSinceEpoch(map['createdAt'] as int),
      updatedAt: DateTime.fromMillisecondsSinceEpoch(map['updatedAt'] as int),
      feedback: map['feedback'] != null ? map['feedback'] as String : null,
      summary: map['summary'] != null ? map['summary'] as String : null,
    );
  }

  factory ClinicalCaseModel.fromJson(String source) {
    return ClinicalCaseModel.fromMap(json.decode(source) as Map<String, dynamic>);
  }

  Map<String, dynamic> toRequestBody() {
    return <String, dynamic>{
      'age': age,
      'sex': sex,
      'type': type.name,
      'pregnant': gestante,
    };
  }

  Map<String, dynamic> toMap() {
    return <String, dynamic>{
      'uid': uid,
      'remoteId': remoteId,
      'threadId': threadId,
      'userId': userId,
      'title': title,
      'type': type.name,
      'age': age,
      'sex': sex,
      'gestante': gestante ? 1 : 0,
      'isReal': isReal ? 1 : 0,
      'anamnesis': anamnesis,
      'physicalExamination': physicalExamination,
      'diagnosticTests': diagnosticTests,
      'finalDiagnosis': finalDiagnosis,
      'management': management,
      'createdAt': createdAt.millisecondsSinceEpoch,
      'updatedAt': updatedAt.millisecondsSinceEpoch,
      'feedback': feedback,
      'summary': summary,
    };
  }
  
  String get textPlane { 
    String text = 'Caso clínico\n';
    text += 'Anamnesis:\n';
    text += anamnesis;
    text += '\nExamen físico:\n';
    text += physicalExamination;
    text += '\nPruebas diagnósticas:\n';
    text += diagnosticTests;
    text += '\nDiagnóstico final:\n';
    text += finalDiagnosis;
    text += '\nManejo:\n';
    text += management;

    return text;
  }

  String get textInMarkDown { 
    String text = '## Caso clínico';
    text += '### anamnesis:';
    text += anamnesis;
    text += '### physicalExamination:';
    text += physicalExamination;
    text += '### diagnosticTests:';
    text += diagnosticTests;
    text += '### finalDiagnosis:';
    text += finalDiagnosis;
    text += '### management:';
    text += management;

    return text;
  }
  
  String toPrompt() {
    String prompt = '**anamnesis:** $anamnesis.';
    prompt += '**physicalExamination:** $physicalExamination.';
    prompt += '**diagnosticTests:** $diagnosticTests.';
    prompt += '**finalDiagnosis:** $finalDiagnosis.';
    prompt += '**management:** $management.';

    return prompt;
  }

  /// Genera un resumen de máximo 2 líneas del caso clínico
  String generateSummary() {
    final words = <String>[];
    
    // Tomar palabras clave de anamnesis (primeras 15 palabras)
    words.addAll(anamnesis.split(' ').take(15));
    
    // Añadir diagnóstico final si está disponible
    if (finalDiagnosis.isNotEmpty) {
      words.add(' - Diagnóstico:');
      words.addAll(finalDiagnosis.split(' ').take(10));
    }
    
    // Limitar a máximo 250 caracteres (aprox. 2 líneas)
    String summary = words.join(' ');
    if (summary.length > 250) {
      summary = summary.substring(0, 247) + '...';
    }
    
    return summary;
  }

  String toJson() => json.encode(toMap());
}
