import 'package:ema_educacion_medica_avanzada/app/clinical_cases/clinical_cases.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_message_model.dart';
import 'package:ema_educacion_medica_avanzada/common/layouts/app_layout.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/outline_ai_button.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/show_error_widget.dart';
import 'package:ema_educacion_medica_avanzada/config/routes/app_pages.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:ema_educacion_medica_avanzada/app/quizzes/widgets/full_feedback_animated.dart';

class ClinicalCaseEvaluationView extends StatefulWidget {
  const ClinicalCaseEvaluationView({super.key});

  @override
  State<ClinicalCaseEvaluationView> createState() =>
      _ClinicalCaseEvaluationViewState();
}

class _ClinicalCaseEvaluationViewState
    extends State<ClinicalCaseEvaluationView> {
  final controller = Get.find<ClinicalCaseController>();
  bool showUserTurns = false;
  ChatMessageModel? evaluationMessage;
  bool isLoadingEvaluation = false;

  Future<void> _loadEvaluationMessage({bool forceReload = false}) async {
    print(
      '[EVAL_LOAD] 🔄 Iniciando carga de evaluación... (forceReload=$forceReload)',
    );

    if (evaluationMessage != null && !forceReload) {
      print('[EVAL_LOAD] ✅ Evaluación ya cargada, saliendo');
      return;
    }
    final caseModel = controller.currentCase.value;
    if (caseModel == null) {
      print('[EVAL_LOAD] ❌ No hay caso actual');
      return;
    }

    print('[EVAL_LOAD] 📋 Caso: ${caseModel.uid} - Tipo: ${caseModel.type}');

    setState(() {
      isLoadingEvaluation = true;
    });

    try {
      print('[EVAL_LOAD] 🔍 Cargando mensajes desde BD...');

      // Cargar todos los mensajes del caso desde la BD
      final allMessagesRaw = await controller.clinicalCaseServive
          .loadMessageByCaseId(caseModel.uid);

      print('[EVAL_LOAD] 📦 Mensajes raw recibidos: ${allMessagesRaw.length}');
      print(
        '[EVAL_LOAD] 📦 Tipo de allMessagesRaw: ${allMessagesRaw.runtimeType}',
      );

      // Casting explícito para asegurar tipo correcto
      final allMessages = List<ChatMessageModel>.from(allMessagesRaw);

      print(
        '[EVAL_LOAD] ✅ Casting exitoso. Total mensajes: ${allMessages.length}',
      );

      final caseType = caseModel.type;

      if (caseType == ClinicalCaseType.analytical) {
        print('[EVAL_LOAD] 🔬 Procesando caso ANALÍTICO');

        // Para analíticos: buscar mensaje con prompt oculto [[HIDDEN_EVAL_PROMPT]]
        // o que contenga 'puntuación' y 'referencias'
        final aiMessages = allMessages.where((m) => m.aiMessage).toList();
        print('[EVAL_LOAD] 🤖 Mensajes AI encontrados: ${aiMessages.length}');

        int idx = 0;
        for (final m in aiMessages.reversed) {
          final lower = m.text.toLowerCase();
          final preview = m.text.substring(
            0,
            m.text.length > 100 ? 100 : m.text.length,
          );
          print('[EVAL_LOAD] 🔎 AI[$idx] - Preview: $preview...');

          // Detección mejorada: buscar indicadores de evaluación
          final hasEvaluationKeywords =
              lower.contains('puntuación') ||
              lower.contains('puntuacion') ||
              lower.contains('resumen clínico') ||
              lower.contains('resumen clinico') ||
              lower.contains('# resumen clínico') || // Markdown header
              lower.contains('# resumen clinico');

          final hasEvaluationSections =
              (lower.contains('desempeño') || lower.contains('desempeno')) &&
              (lower.contains('fortalezas') ||
                  lower.contains('áreas de mejora'));

          final hasHiddenPrompt = m.text.contains('[[HIDDEN_EVAL_PROMPT]]');

          // Evaluación típicamente >1500 chars, mensajes de chat <1500 chars
          final isLongEnough = m.text.length > 1500;

          if (hasHiddenPrompt ||
              hasEvaluationKeywords ||
              (hasEvaluationSections && isLongEnough)) {
            evaluationMessage = m;
            print('[EVAL_LOAD] ✅ Evaluación encontrada en AI[$idx]');
            print('[EVAL_LOAD] 📝 Longitud del texto: ${m.text.length} chars');
            print(
              '[EVAL_LOAD] 📝 Razón: ${hasHiddenPrompt
                  ? "HIDDEN_PROMPT"
                  : hasEvaluationSections
                  ? "SECTIONS"
                  : "KEYWORDS"}',
            );
            break;
          }
          idx++;
        }
        // Fallback: último mensaje AI
        if (evaluationMessage == null) {
          evaluationMessage = aiMessages.lastOrNull;
          print('[EVAL_LOAD] ⚠️ Usando fallback: último mensaje AI');
        }
      } else {
        print('[EVAL_LOAD] 🎮 Procesando caso INTERACTIVO');

        // Interactivo: buscar mensaje con 'Resumen Final:'
        final aiMessages = allMessages.where((m) => m.aiMessage).toList();
        print('[EVAL_LOAD] 🤖 Mensajes AI encontrados: ${aiMessages.length}');

        for (final m in aiMessages.reversed) {
          final lower = m.text.toLowerCase();
          if (lower.startsWith('resumen final:')) {
            evaluationMessage = m;
            print('[EVAL_LOAD] ✅ Evaluación interactiva encontrada');
            break;
          }
          if (lower.startsWith('evaluación final interactiva')) {
            continue; // ignorar duplicado
          }
        }
        if (evaluationMessage == null) {
          evaluationMessage = aiMessages.lastOrNull;
          print('[EVAL_LOAD] ⚠️ Usando fallback: último mensaje AI');
        }
      }

      if (evaluationMessage != null) {
        print('[EVAL_LOAD] ✅ Evaluación cargada exitosamente');
        print('[EVAL_LOAD] 📊 UID: ${evaluationMessage!.uid}');
        print('[EVAL_LOAD] 📊 Format: ${evaluationMessage!.format}');
      } else {
        print('[EVAL_LOAD] ❌ No se encontró mensaje de evaluación');
      }
    } catch (e, stackTrace) {
      print('[EVAL_LOAD] ❌ ERROR: $e');
      print('[EVAL_LOAD] 📚 StackTrace: $stackTrace');
    } finally {
      setState(() {
        isLoadingEvaluation = false;
      });
      print('[EVAL_LOAD] 🏁 Finalizando carga (isLoading=false)');
    }
  }

  // Método legacy para casos INTERACTIVOS: obtener de controller.messages
  ChatMessageModel? _getInteractiveEvaluationFromMessages() {
    final msgs = controller.messages.where((m) => m.aiMessage).toList();
    if (msgs.isEmpty) return null;

    // Interactivo: priorizar mensaje con 'Resumen Final:'
    ChatMessageModel? resumen;
    for (final m in msgs.reversed) {
      final lower = m.text.toLowerCase();
      if (resumen == null && lower.startsWith('resumen final:')) {
        resumen = m;
      }
      if (lower.startsWith('evaluación final interactiva')) {
        continue; // ignorar duplicado
      }
      if (resumen != null) break;
    }
    return resumen ?? msgs.last;
  }

  @override
  void initState() {
    super.initState();

    final caseType = controller.currentCase.value?.type;

    // Solo para casos ANALÍTICOS: cargar evaluación desde BD
    if (caseType == ClinicalCaseType.analytical) {
      _loadEvaluationMessage();

      if (!controller.evaluationGenerated.value) {
        controller.generateFinalEvaluation().then((_) {
          // Recargar el mensaje después de generar la evaluación (forzar)
          _loadEvaluationMessage(forceReload: true);
        });
      }

      // Listener para recargar cuando se genere la evaluación (solo analíticos)
      ever(controller.evaluationGenerated, (generated) {
        if (generated &&
            controller.currentCase.value?.type == ClinicalCaseType.analytical) {
          _loadEvaluationMessage(forceReload: true);
        }
      });
    }
    // Para casos INTERACTIVOS: usar el flujo existente (controller.messages)
  }

  // (Se removieron heurísticas de métricas de resultados; el diseño ahora depende del feedback generado)

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
        final canShowEval =
            caseModel.type == ClinicalCaseType.analytical ||
            controller.interactiveEvaluationGenerated.value;
        if (!canShowEval && isInteractive && controller.isComplete.value) {
          // Mostrar pantalla con botón para revelar
          return _buildRevealButton(context, hasHidden);
        }

        // Obtener mensaje de evaluación según el tipo de caso
        ChatMessageModel? evalMsg;
        bool isLoading = false;

        if (caseModel.type == ClinicalCaseType.analytical) {
          // Analíticos: cargar desde BD (ya cargado en initState)
          evalMsg = evaluationMessage;
          isLoading = isLoadingEvaluation;
        } else {
          // Interactivos: obtener desde messages (flujo original)
          evalMsg = _getInteractiveEvaluationFromMessages();
          isLoading = false;
        }

        // Mostrar loader mientras se carga la evaluación (solo para analíticos)
        if (isLoading || evalMsg == null) {
          return const StateMessageWidget(
            message: 'Cargando evaluación...',
            type: StateMessageType.download,
            showLoading: true,
          );
        }

        return SingleChildScrollView(
          padding: const EdgeInsets.symmetric(horizontal: 28, vertical: 32),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(
                caseModel.type == ClinicalCaseType.interactive
                    ? 'Evaluación final interactiva'
                    : 'Evaluación final analítica',
                style: Theme.of(context).textTheme.headlineSmall?.copyWith(
                  fontWeight: FontWeight.bold,
                  color: Theme.of(context).primaryColor,
                ),
              ),
              const SizedBox(height: 12),
              Text(
                caseModel.anamnesis,
                style: Theme.of(context).textTheme.bodySmall,
                maxLines: 3,
                overflow: TextOverflow.ellipsis,
              ),
              const Divider(height: 32),
              // Feedback con mismo formato y orden que cuestionarios
              FullFeedbackAnimated(
                fitGlobal: evalMsg.text,
                questions: controller.questions.toList(),
                animate: false,
                renderMarkdown: true,
              ),
              const SizedBox(height: 24),
              const SizedBox(height: 16),
              Row(
                children: [
                  Expanded(
                    child: OutlineAiButton(
                      text: 'Ver intervenciones',
                      onPressed: () => _showUserTurnsBottomSheet(context),
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
            Text(
              'Evaluación final disponible',
              style: Theme.of(context).textTheme.titleLarge,
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 12),
            Text(
              'Pulsa el botón para ver tu evaluación final del caso interactivo.',
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodyMedium,
            ),
            const SizedBox(height: 32),
            OutlineAiButton(
              text: 'Ver evaluación final',
              onPressed:
                  hasHidden
                      ? () async {
                        await controller.showInteractiveSummaryIfAvailable();
                        setState(() {});
                      }
                      : null,
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

  void _showUserTurnsBottomSheet(BuildContext context) async {
    final caseId = controller.currentCase.value?.uid;
    final caseModel = controller.currentCase.value;

    if (caseModel == null) return;

    // Cargar desde BD si es caso analítico, de messages si es interactivo
    final turns =
        caseModel.type == ClinicalCaseType.analytical
            ? await controller.getUserInterventionsFromDb(caseId!)
            : controller.messages
                .where((m) => !m.aiMessage && m.chatId == caseId)
                .toList();

    if (!context.mounted) return;
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (ctx) {
        return SafeArea(
          child: Padding(
            padding: const EdgeInsets.fromLTRB(16, 12, 16, 24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Expanded(
                      child: Text(
                        'Intervenciones del usuario',
                        style: Theme.of(ctx).textTheme.titleMedium?.copyWith(
                          fontWeight: FontWeight.bold,
                          color: Theme.of(ctx).primaryColor,
                        ),
                      ),
                    ),
                    IconButton(
                      icon: const Icon(Icons.close),
                      onPressed: () => Navigator.of(ctx).pop(),
                    ),
                  ],
                ),
                const SizedBox(height: 8),
                if (turns.isEmpty)
                  Text(
                    'Sin intervenciones registradas',
                    style: Theme.of(ctx).textTheme.bodySmall,
                  )
                else
                  Flexible(
                    child: ListView.separated(
                      shrinkWrap: true,
                      itemCount: turns.length,
                      separatorBuilder: (_, __) => const Divider(height: 16),
                      itemBuilder:
                          (_, i) => Text(
                            '${i + 1}. ${turns[i].text}',
                            textAlign: TextAlign.justify,
                            style: Theme.of(ctx).textTheme.bodyMedium,
                          ),
                    ),
                  ),
              ],
            ),
          ),
        );
      },
    );
  }
}
