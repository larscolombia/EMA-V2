part of 'app_overlays.dart';


enum OverlayRoutes {
  clinicalCaseAnlytical,
  clinicalCaseInteractive,
  // clinicalCasePreview,
  // clinicalCasesHistory,
  getPremium,
  homeClinicalCasesMenu,
  homeQuizzesMenu,
  // pdfSearchList,
  pdfUpdloader,
  quizzesGeneral,
  // quizzesHistory,
  quizzesSpeciality,
  // quizzPreview,
  empty;

  String get name {
    return toString();
  }
} 
