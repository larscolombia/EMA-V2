// import 'package:ema_educacion_medica_avanzada/common/bindings.dart';
import 'package:ema_educacion_medica_avanzada/app/categories/categories.dart';
import 'package:ema_educacion_medica_avanzada/app/clinical_cases/clinical_cases.dart';
import 'package:ema_educacion_medica_avanzada/app/pdf/views/file_uploader_tabs.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/quizzes.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:flutter/widgets.dart';

part 'overlay_routes.dart';

class AppOverlays {
  static final List<EmaOverlayRoute> overlayRoutes = [
    // clinicalCaseAnlytical - OK
    EmaOverlayRoute(
      name: OverlayRoutes.clinicalCaseAnlytical.name,
      topArea:
          () => OverlayTitle(
            title: 'Casos Clínicos Analíticos',
            subtitle:
                'Explora casos clínicos completos con información detallada para análisis del procedimiento.',
          ),
      bottomArea: () => ClinicalCaseOptions(type: ClinicalCaseType.analytical),
    ),

    // clinicalCaseInteractive - OK
    EmaOverlayRoute(
      name: OverlayRoutes.clinicalCaseInteractive.name,
      topArea:
          () => OverlayTitle(
            title: 'Casos Clínicos Interactivos',
            subtitle:
                'Interactúa con casos clínicos simulados, toma decisiones y recibe retroalimentación en tiempo real.',
          ),
      bottomArea: () => ClinicalCaseOptions(type: ClinicalCaseType.interactive),
    ),

    // clinicalCasePreview
    // EmaOverlayRoute(
    //   name: OverlayRoutes.clinicalCasePreview.name,
    //   bottomArea: () => Text('clinicalCasePreview -- Bottom Child'),
    //   topArea: () => Text('clinicalCasePreview -- Top Child'),
    // ),

    // clinicalCasesHistory
    // EmaOverlayRoute(
    //   name: OverlayRoutes.clinicalCasesHistory.name,
    //   bottomArea: () => Text('clinicalCasesHistory -- Bottom Child'),
    //   topArea: () => Text('clinicalCasesHistory -- Top Child'),
    // ),

    // getPremium
    EmaOverlayRoute(
      name: OverlayRoutes.getPremium.name,
      bottomArea: () => Text('getPremium -- Bottom Child'),
      topArea: () => Text('getPremium -- Top Child'),
    ),

    // homeClinicalCasesMenu
    EmaOverlayRoute(
      name: OverlayRoutes.homeClinicalCasesMenu.name,
      topArea:
          () => OverlayTitle(
            title: 'Casos Clínicos',
            subtitle:
                'Por a prueba tus conocimientos médicos con evaluaciones diseñadas para mejorar tus habilidades clínicas.',
          ),
      bottomArea: () => ClinicalCasesMenu(),
    ),

    // homeQuizzesMenu
    EmaOverlayRoute(
      name: OverlayRoutes.homeQuizzesMenu.name,
      topArea:
          () => OverlayTitle(
            title: 'Cuestionarios Médicos',
            subtitle:
                'Pon a prueba tus conocimientos médicos con evaluaciones diseñadas para mejorar tus habilidades clínicas.',
          ),
      bottomArea: () => QuizzesMenu(),
    ),

    // pdfSearchList
    // EmaOverlayRoute(
    //   name: OverlayRoutes.pdfSearchList.name,
    //   bottomArea: () => Text('pdfSearchList -- Bottom Child'),
    //   topArea: () => Text('pdfSearchList -- Top Child'),
    // ),

    // pdfUpdloader
    EmaOverlayRoute(
      name: OverlayRoutes.pdfUpdloader.name,
      bottomArea: () => const FileUploaderTabs(),
    ),

    // quizzesGeneral - OK
    EmaOverlayRoute(
      name: OverlayRoutes.quizzesGeneral.name,
      topArea:
          () => OverlayTitle(
            title: 'Cuestionarios Médicos',
            subtitle:
                'Pon a prueba tus conocimientos médicos con evaluaciones diseñadas para mejorar tus habilidades clínicas.',
          ),
      bottomArea: () => QuizzesOptions(withCategory: false),
    ),

    // quizzesHistory
    // EmaOverlayRoute(
    //   name: OverlayRoutes.quizzesHistory.name,
    //   bottomArea: () => Text('quizzesHistory -- Bottom Child'),
    //   topArea: () => Text('quizzesHistory -- Top Child'),
    // ),

    // quizzesSpeciality - OK
    EmaOverlayRoute(
      name: OverlayRoutes.quizzesSpeciality.name,
      topList: () => CategoriesOptionsList(),
      bottomArea: () => QuizzesOptions(withCategory: true),
    ),

    // quizzPreview
    // EmaOverlayRoute(
    //   name: OverlayRoutes.quizzPreview.name,
    //   bottomArea: () => Text('quizzPreview -- Bottom Child'),
    //   topArea: () => Text('quizzPreview -- Top Child'),
    // ),

    // empty
    EmaOverlayRoute(
      name: OverlayRoutes.empty.name,
      bottomArea: () => Text('empty -- Bottom Child'),
      topArea: () => Text('empty -- Top Child'),
    ),
  ];
}
