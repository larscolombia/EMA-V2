import 'package:ema_educacion_medica_avanzada/app/clinical_cases/clinical_cases.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_message_model.dart';
import 'package:ema_educacion_medica_avanzada/common/layouts/app_layout.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/outline_ai_button.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/show_error_widget.dart';
import 'package:ema_educacion_medica_avanzada/config/routes/app_pages.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:gpt_markdown/gpt_markdown.dart';

class ClinicalCaseEvaluationView extends StatefulWidget {
  const ClinicalCaseEvaluationView({super.key});

  @override
  State<ClinicalCaseEvaluationView> createState() => _ClinicalCaseEvaluationViewState();
}

class _ClinicalCaseEvaluationViewState extends State<ClinicalCaseEvaluationView> {
  final controller = Get.find<ClinicalCaseController>();
  bool showUserTurns = false;

  ChatMessageModel? _evaluationMessage() {
    final msgs = controller.messages.where((m) => m.aiMessage).toList();
    if (msgs.isEmpty) return null;
    final caseType = controller.currentCase.value?.type;
    if (caseType == ClinicalCaseType.analytical) {
      for (final m in msgs.reversed) {
        final lower = m.text.toLowerCase();
        if (lower.contains('puntuación') && lower.contains('referencias')) {
          return m;
        }
      }
      return msgs.last;
    } else {
      // Interactivo: priorizar mensaje con 'Resumen Final:' si existe; evitar el que inicie con 'Evaluación final interactiva' (duplicado)
      ChatMessageModel? resumen;
      for (final m in msgs.reversed) {
        final lower = m.text.toLowerCase();
        if (resumen == null && lower.startsWith('resumen final:')) {
          resumen = m; // preferido
        }
        if (lower.startsWith('evaluación final interactiva')) {
          // ignorar duplicado generado por capa legacy
          continue;
        }
        if (resumen != null) break;
      }
      return resumen ?? msgs.last;
    }
  }

  @override
  void initState() {
    super.initState();
    if (!controller.evaluationGenerated.value && controller.currentCase.value?.type == ClinicalCaseType.analytical) {
      controller.generateFinalEvaluation();
    }
  }

  Widget _interactiveDeterministicStats() {
    // Construir métricas desde las preguntas (determinismo local)
    final caseType = controller.currentCase.value?.type;
    if (caseType != ClinicalCaseType.interactive) return const SizedBox.shrink();
    final answered = controller.questions.where((q) => q.userAnswer != null && q.userAnswer!.isNotEmpty).toList();
    int correct = 0;
    for (final q in answered) {
      final ans = q.answer ?? '';
      final user = q.userAnswer ?? '';
      if (ans.isNotEmpty && _answerIsCorrect(user, ans)) correct++;
    }
    final total = answered.length;
    final pct = total == 0 ? 0 : (correct * 100 / total).round();
    if (total == 0) return const SizedBox.shrink();
    Color badgeColor;
    if (pct >= 80) {
      badgeColor = Colors.green; 
    } else if (pct >= 50) {
      badgeColor = Colors.orange; 
    } else { 
      badgeColor = Colors.red; 
    }
    return Container(
      width: double.infinity,
      margin: const EdgeInsets.only(top: 16, bottom: 8),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: badgeColor.withOpacity(0.08),
        border: Border.all(color: badgeColor.withOpacity(0.4)),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(Icons.assessment_outlined, color: badgeColor),
              const SizedBox(width: 8),
              Text('Resultado local (determinista)', style: TextStyle(fontWeight: FontWeight.bold, color: badgeColor)),
            ],
          ),
          const SizedBox(height: 8),
          Text('Correctas: $correct / $total  ·  $pct%', style: Theme.of(context).textTheme.bodyMedium),
        ],
      ),
    );
  }

  // Heurísticas de normalización y similitud (paridad con backend)
  String _stripLeadingLetterPrefix(String s) {
    final trim = s.trim();
    if (trim.length >= 3) {
      final c = trim.codeUnitAt(0);
      if ((c >= 'A'.codeUnitAt(0) && c <= 'D'.codeUnitAt(0)) || (c >= 'a'.codeUnitAt(0) && c <= 'd'.codeUnitAt(0))) {
        final rest = trim.substring(1).trimLeft();
        if (rest.startsWith('-') || rest.startsWith(')')) {
          return rest.substring(1).trim();
        }
      }
    }
    return trim;
  }

  String _normalizeAnswer(String s) {
    if (s.isEmpty) return '';
    var out = s.trim().toLowerCase();
    // Replace common accents -> simple map
    const accents = {
      'á': 'a', 'é': 'e', 'í': 'i', 'ó': 'o', 'ú': 'u', 'ü': 'u', 'ñ': 'n'
    };
    accents.forEach((k, v) {
      out = out.replaceAll(k, v);
    });
    // remove punctuation, keep letters/numbers and spaces
    out = out.replaceAll(RegExp(r"[^\p{L}\p{N}\s]", unicode: true), ' ');
    // collapse spaces
    out = out.replaceAll(RegExp(r"\s+"), ' ').trim();
    return out;
  }

  List<String> _tokenize(String s) {
    if (s.isEmpty) return [];
    return s.split(RegExp(r"\s+")).where((e) => e.isNotEmpty).toList();
  }

  double _jaccard(List<String> a, List<String> b) {
    if (a.isEmpty || b.isEmpty) return 0.0;
    final setA = <String>{};
    final setB = <String>{};
    for (final w in a) setA.add(w);
    for (final w in b) setB.add(w);
    var inter = 0;
    for (final w in setA) if (setB.contains(w)) inter++;
    final un = setA.length + setB.length - inter;
    if (un == 0) return 0.0;
    return inter / un;
  }

  bool _answerIsCorrect(String userRaw, String correctRaw) {
    final userTrim = userRaw.trim();
    // If either empty -> false
    if (userTrim.isEmpty || correctRaw.trim().isEmpty) return false;
    // strip leading letter prefixes like 'A - '
    final userClean = _normalizeAnswer(_stripLeadingLetterPrefix(userRaw));
    final correctClean = _normalizeAnswer(_stripLeadingLetterPrefix(correctRaw));
    if (userClean.isEmpty || correctClean.isEmpty) return false;
    if (userClean == correctClean) return true;
    final ut = _tokenize(userClean);
    final ct = _tokenize(correctClean);
    if (ut.length >= 2) {
      final jac = _jaccard(ut, ct);
      if (jac >= 0.8) return true;
      // subset allowance
      var subset = true;
      for (final w in ut) {
        if (!ct.contains(w)) { subset = false; break; }
      }
      if (subset && (ut.length / (ct.isEmpty ? 1 : ct.length)) >= 0.5) return true;
    }
    if (userClean.length >= 15 && correctClean.contains(userClean)) return true;
    return false;
  }

  @override
  Widget build(BuildContext context) {
    return AppLayout(
      backRoute: Routes.home.name,
      body: Obx(() {
        final caseModel = controller.currentCase.value;
        if (caseModel == null) {
          return const StateMessageWidget(
            message: 'Caso no disponible',
            type: StateMessageType.noSearchResults,
          );
        }
        final isInteractive = caseModel.type == ClinicalCaseType.interactive;
        final hasHidden = controller.hasHiddenInteractiveSummary;
        // Evaluación mostrada solo si ya se reveló (interactiveEvaluationGenerated) o es analítico
        final canShowEval = caseModel.type == ClinicalCaseType.analytical || controller.interactiveEvaluationGenerated.value;
        if (!canShowEval && isInteractive && controller.isComplete.value) {
          // Mostrar pantalla con botón para revelar
          return _buildRevealButton(context, hasHidden);
        }
        final evalMsg = _evaluationMessage();
        if (evalMsg == null) {
          return const StateMessageWidget(
            message: 'No se encontró la evaluación generada.',
            type: StateMessageType.noSearchResults,
          );
        }
        return SingleChildScrollView(
          padding: const EdgeInsets.symmetric(horizontal: 28, vertical: 32),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                controller.currentCase.value?.type == ClinicalCaseType.interactive
                    ? 'Evaluación final interactiva'
                    : 'Evaluación final analítica',
                style: Theme.of(context).textTheme.headlineSmall?.copyWith(fontWeight: FontWeight.bold),
              ),
              const SizedBox(height: 12),
              Text(caseModel.anamnesis, style: Theme.of(context).textTheme.bodySmall, maxLines: 3, overflow: TextOverflow.ellipsis),
              const Divider(height: 32),
              if (caseModel.type == ClinicalCaseType.interactive) _interactiveDeterministicStats(),
              GptMarkdown(evalMsg.text),
              const SizedBox(height: 32),
              AnimatedCrossFade(
                firstChild: const SizedBox.shrink(),
                secondChild: _buildUserTurns(context),
                crossFadeState: showUserTurns ? CrossFadeState.showSecond : CrossFadeState.showFirst,
                duration: const Duration(milliseconds: 250),
              ),
              const SizedBox(height: 16),
              Row(
                children: [
                  Expanded(
                    child: OutlineAiButton(
                      text: showUserTurns ? 'Ocultar intervenciones' : 'Ver intervenciones',
                      onPressed: () => setState(() => showUserTurns = !showUserTurns),
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: OutlineAiButton(
                      text: 'Inicio',
                      onPressed: () => Get.offAllNamed(Routes.home.name),
                    ),
                  ),
                ],
              ),
            ],
          ),
        );
      }),
    );
  }

  Widget _buildRevealButton(BuildContext context, bool hasHidden) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 32),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Icon(Icons.lock_outline, size: 52),
            const SizedBox(height: 24),
            Text('Evaluación final disponible', style: Theme.of(context).textTheme.titleLarge, textAlign: TextAlign.center),
            const SizedBox(height: 12),
            Text('Pulsa el botón para ver tu evaluación final del caso interactivo.', textAlign: TextAlign.center, style: Theme.of(context).textTheme.bodyMedium),
            const SizedBox(height: 32),
            OutlineAiButton(
              text: 'Ver evaluación final',
              onPressed: hasHidden ? () async {
                await controller.showInteractiveSummaryIfAvailable();
                setState(() {});
              } : null,
              isLoading: false,
            ),
            const SizedBox(height: 16),
            OutlineAiButton(
              text: 'Inicio',
              onPressed: () => Get.offAllNamed(Routes.home.name),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildUserTurns(BuildContext context) {
    final caseId = controller.currentCase.value?.uid;
    final turns = controller.messages.where((m) => !m.aiMessage && m.chatId == caseId).toList();
    if (turns.isEmpty) {
      return Text('Sin intervenciones de usuario registradas', style: Theme.of(context).textTheme.bodySmall);
    }
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surfaceVariant.withOpacity(0.3),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('Intervenciones del usuario', style: Theme.of(context).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
          const SizedBox(height: 8),
          ...List.generate(turns.length, (i) {
            final t = turns[i];
            return Padding(
              padding: const EdgeInsets.only(bottom: 6),
              child: Text('${i + 1}. ${t.text}', style: Theme.of(context).textTheme.bodySmall),
            );
          }),
        ],
      ),
    );
  }
}
