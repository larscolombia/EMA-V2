// ignore_for_file: public_member_api_docs, sort_constructors_first

import 'dart:convert';

import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_message_model.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/models/chat_model.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/services/chats_service.dart';
import 'package:ema_educacion_medica_avanzada/core/attachments/attachment_service.dart';
import 'package:ema_educacion_medica_avanzada/core/attachments/pdf_attachment.dart';
import 'package:ema_educacion_medica_avanzada/core/attachments/image_attachment.dart';
import 'package:ema_educacion_medica_avanzada/core/logger/logger.dart';
import 'package:ema_educacion_medica_avanzada/core/notify/notify.dart';
import 'package:ema_educacion_medica_avanzada/core/ui/ui_observer_service.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:ema_educacion_medica_avanzada/core/preferences/string_preference.dart';
import 'package:ema_educacion_medica_avanzada/core/users/user_service.dart';

import '../../../config/config.dart';
import '../../profiles/profiles.dart';

class ChatController extends GetxService {
  ScrollController? _chatScrollController;
  FocusNode? _focusNode;

  final attachmentService = Get.find<AttachmentService>();
  final chatsService = Get.find<ChatsService>();
  final keyboardService = Get.find<UiObserverService>();
  final profileController = Get.find<ProfileController>();
  final userService = Get.find<UserService>();

  final currentChat = ChatModel.empty().obs;
  final messages = <ChatMessageModel>[].obs;
  final loading = false.obs;
  final error = ''.obs;
  final isTyping = false.obs;
  final isSending = false.obs;
  // Current backend processing stage (driven by SSE markers)
  final currentStage =
      ''.obs; // values like: start, rag_search, rag_found, rag_empty, pubmed_search, pubmed_found, streaming_answer

  // Evita llamadas duplicadas a /conversations/start durante la creación del primer mensaje (migrado desde /asistente/start)
  bool _startingNewChat = false;

  String threadId = '';
  final threadPref = StringPreference(
    key: 'current_thread_id',
    defaultValue: '',
  );

  final Rx<PdfAttachment?> pendingPdf = Rx<PdfAttachment?>(null);
  final Rx<ImageAttachment?> pendingImage = Rx<ImageAttachment?>(null);
  final isUploadingPdf = false.obs;
  final isUploadingImage = false.obs;
  final userScrolling = false.obs;

  // Control para modo focus en PDF específico
  final focusOnPdfMode = false.obs;
  String? focusedPdfId;

  final userManuallyScrolled = false.obs;
  final autoScrollEnabled = true.obs;

  // Snapshot of last failed send to enable retry
  ChatMessageModel? _lastFailedUserMessage;
  PdfAttachment? _lastFailedPdf;

  // App lifecycle tracking for recovery
  final isAppResumed = false.obs;
  final lastSendTime = DateTime.now().obs;

  /// Helper para procesar tokens SSE, incluyendo el evento final __JSON__ con texto completo bien formateado
  void _processStreamToken(
    String token,
    ChatMessageModel aiMessage,
    bool Function() hasFirstToken,
    void Function(bool) setHasFirstToken,
  ) {
    if (token.startsWith('__STAGE__:')) {
      final stage = token.split(':').last.trim();
      currentStage.value = stage;
      return;
    }

    // CRÍTICO: Detectar evento final con JSON completo bien formateado
    if (token.startsWith('__JSON__:')) {
      try {
        final jsonStr = token.substring(9); // Quitar "__JSON__:"
        final jsonData = json.decode(jsonStr) as Map<String, dynamic>;
        var finalText = jsonData['text'] as String;

        // CRÍTICO: Limpiar cualquier marcador __STAGE__: que pudiera haber quedado
        // (por si el backend los incluyó antes del fix)
        finalText = finalText.replaceAll(RegExp(r'__STAGE__:[^\s]+'), '').trim();
        
        // Limpiar múltiples saltos de línea que puedan haber quedado
        finalText = finalText.replaceAll(RegExp(r'\n{3,}'), '\n\n');

        // Reemplazar texto acumulado con versión final bien formateada
        aiMessage.text = finalText;
        messages.refresh();
        scrollToBottom();

        print(
          '✅ [Controller] Final JSON received: ${finalText.length} chars, ${jsonData['newline_count']} newlines',
        );
        print(
          '✅ [Controller] Preview: "${finalText.substring(0, finalText.length > 200 ? 200 : finalText.length).replaceAll('\n', '\\n')}"',
        );
        return;
      } catch (e) {
        print('❌ [Controller] Error parsing final JSON: $e');
        // NO mostrar el JSON literal en la UI si falla el parsing
        return;
      }
    }

    if (!hasFirstToken()) {
      setHasFirstToken(true);
      aiMessage.text = token;
      if (!messages.contains(aiMessage)) {
        messages.add(aiMessage);
      }
    } else {
      aiMessage.text += token;
    }
    messages.refresh();
    scrollToBottom();
  }

  @override
  void onInit() {
    super.onInit();
    threadPref.getValue().then((value) {
      threadId = value;
      // debug: persisted thread id recovered
      Logger.debug('threadId recovered prefs="$threadId"');
      // Ya no forzamos reset inmediato: permitimos reanudar conversación existente.
      // Si más adelante se detecta inconsistencia (por ejemplo, backend responde 404),
      // se limpiará explícitamente en el flujo de error de startChat o sendMessage.
      if (threadId.isNotEmpty) {
        Logger.debug('resume threadId=$threadId');
      }
    });
    keyboardService.isKeyboardVisible.listen((visible) {
      if (visible && messages.isNotEmpty && autoScrollEnabled.value) {
        Future.delayed(const Duration(milliseconds: 150), () {
          scrollToBottom();
        });
      }
    });

    // Listen for app lifecycle changes to detect when app resumes
    _setupAppLifecycleListener();
  }

  void _setupAppLifecycleListener() {
    // TODO: Implement with WidgetsBindingObserver if needed
    // For now, we'll detect stuck states in sendMessage
  }

  void _checkForStuckState() {
    // If we're "sending" for more than 60 seconds, something went wrong
    if (isSending.value || isTyping.value) {
      final timeSinceLastSend = DateTime.now().difference(lastSendTime.value);
      if (timeSinceLastSend.inSeconds > 60) {
        Logger.warn('stuck state (>60s) resetting');
        _resetSendingState();

        // Show retry option to user
        if (_lastFailedUserMessage != null) {
          messages.add(
            ChatMessageModel.temporal(
              '¡Ups! Parece que la respuesta se quedó pendiente. '
              'Puedes intentar nuevamente.',
              true,
            ),
          );
          scrollToBottom();
        }
      }
    }
  }

  /// Force stop current operation and reset state
  void forceStopAndReset() {
    Logger.warn('forceStopAndReset invoked');
    _resetSendingState();

    // Remove any loading/typing indicators from messages
    if (messages.isNotEmpty &&
        messages.last.aiMessage &&
        messages.last.text.isEmpty) {
      messages.removeLast();
    }

    // Show user they can retry
    messages.add(
      ChatMessageModel.temporal(
        'Operación detenida. Puedes intentar enviar tu mensaje nuevamente.',
        true,
      ),
    );
    scrollToBottom();
  }

  void _resetSendingState() {
    isSending.value = false;
    isTyping.value = false;
    isUploadingPdf.value = false;
  }

  void cleanChat() async {
    currentChat.value = ChatModel.empty();
    messages.value = [];
    pendingPdf.value = null;
    pendingImage.value = null;
    error.value = '';
    threadId = '';
    threadPref.setValue('');
    // Limpiar estado del focus mode para nuevo chat
    focusOnPdfMode.value = false;
    focusedPdfId = null;
  }

  Future<void> _loadChatById(String chatId) async {
    try {
      final chat = await chatsService.getChatById(chatId);

      if (chat != null) {
        currentChat.value = chat;
        threadId = chat.threadId;
        threadPref.setValue(threadId);
      } else {
        throw Exception('Chat no encontrado con ID: $chatId');
      }
    } catch (e) {
      print('❌ [ChatController] Error en _loadChatById: $e');
      rethrow; // Re-lanzar para que showChat lo capture
    }
  }

  Future<void> _loadMessagesByChatId(String chatId) async {
    try {
      loading.value = true;

      final chatMessages = await chatsService.getMessagesById(chatId);

      if (chatMessages.isEmpty) {
        print(
          '⚠️ [ChatController] No se encontraron mensajes para el chat: $chatId',
        );
      }

      messages.value = chatMessages;

      scrollToBottom();

      loading.value = false;
    } catch (e) {
      print('❌ [ChatController] Error en _loadMessagesByChatId: $e');
      error.value = 'Error al cargar mensajes';
      loading.value = false;
      rethrow; // Re-lanzar para que showChat lo capture
    }
  }

  Future<void> startNewChat() async {
    try {
      cleanChat();
      isTyping.value = true;
      _startingNewChat = true;
      Logger.debug('startNewChat -> startChat("")');
      final start = await chatsService.startChat('');
      // En modo test forzamos un threadId estable para validar persistencia
      threadId =
          Get.testMode
              ? (start.threadId.isNotEmpty ? start.threadId : 'test-thread')
              : start.threadId;
      threadPref.setValue(threadId);

      messages.add(
        ChatMessageModel.ai(chatId: currentChat.value.uid, text: start.text),
      );
      scrollToBottom();
    } catch (e) {
      error.value = 'Error al iniciar conversación';
    } finally {
      _startingNewChat = false;
      isTyping.value = false;
    }
  }

  void attachPdf(PdfAttachment pdf) {
    pendingPdf.value = pdf;
    // Cuando se adjunta un PDF, activar automáticamente el modo focus
    focusOnPdfMode.value = true;
    focusedPdfId = pdf.uid;
  }

  void attachImage(ImageAttachment image) {
    pendingImage.value = image;
  }

  void toggleFocusOnPdf() {
    focusOnPdfMode.value = !focusOnPdfMode.value;
    if (!focusOnPdfMode.value) {
      focusedPdfId = null;
    } else if (pendingPdf.value != null) {
      focusedPdfId = pendingPdf.value!.uid;
    }
  }

  /// Resetea completamente el estado del chat para conversación completamente nueva
  void forceNewConversation() {
    cleanChat();
    // Forzar nuevo threadId
    threadId = '';
    threadPref.setValue('');
    print('🆕 [ChatController] Forzando nueva conversación - thread limpiado');
  }

  /// Retry the last failed send without duplicating the user bubble.
  Future<void> retryLastSend() async {
    if (isSending.value) return;
    final failedMsg = _lastFailedUserMessage;
    final failedPdf = _lastFailedPdf;
    if (failedMsg == null) return;

    // Remove the last error bubble if it is present
    if (messages.isNotEmpty &&
        messages.last.aiMessage &&
        (messages.last.text.contains('¡Ups!') ||
            messages.last.text.contains('Error'))) {
      messages.removeLast();
    }

    isSending.value = true;
    isTyping.value = true;
    try {
      final aiMessage = ChatMessageModel.ai(chatId: failedMsg.chatId, text: '');
      var hasFirstToken = false;
      final response = await chatsService.sendMessage(
        threadId: threadId,
        userMessage: failedMsg,
        file: failedPdf,
        onStream: (token) {
          _processStreamToken(
            token,
            aiMessage,
            () => hasFirstToken,
            (val) => hasFirstToken = val,
          );
        },
      );
      if (!hasFirstToken) {
        aiMessage.text = response.text;
        messages.add(aiMessage);
      } else {
        // Don't overwrite the streamed content with response.text!
        // aiMessage.text = response.text;
      }

      // Success: clear snapshot
      _lastFailedUserMessage = null;
      _lastFailedPdf = null;
      // File quota now consumed server-side (file_upload flow); no client decrement.
    } catch (e) {
      messages.add(
        ChatMessageModel.temporal(
          '¡Ups! Parece que hubo un problema al procesar tu mensaje. '
          'Por favor, intenta nuevamente en unos momentos.',
          true,
        ),
      );
      scrollToBottom();
    } finally {
      isSending.value = false;
      isTyping.value = false;
    }
  }

  Future<void> sendMessage([String? userText]) async {
    final String cleanUserText = userText != null ? userText.trim() : '';
    final PdfAttachment? currentPdf = pendingPdf.value;
    final ImageAttachment? currentImage = pendingImage.value;

    if (cleanUserText.isEmpty && currentPdf == null && currentImage == null)
      return;
    if (isSending.value) {
      // Check if we're stuck before rejecting
      _checkForStuckState();
      return;
    }

    // Si no hay PDF actual, desactivar focus mode automáticamente
    if (currentPdf == null && focusOnPdfMode.value) {
      print('🔄 [ChatController] Desactivando focus mode - sin PDF');
      focusOnPdfMode.value = false;
      focusedPdfId = null;
    }

    // Separar texto para UI vs backend desde el inicio
    // Si el usuario envía solo el PDF sin texto, el backend se encargará
    // de generar automáticamente el prompt estructurado con STRUCTURED_PDF_SUMMARY=1
    var effectiveText = cleanUserText;
    var displayText = cleanUserText; // Texto que se muestra en la UI

    if (effectiveText.isEmpty && currentPdf != null) {
      // Para PDFs sin texto, enviar prompt vacío al backend
      // El backend generará automáticamente el prompt estructurado
      effectiveText = '';
      // Texto simple para mostrar en la UI (vacío = solo PDF)
      displayText = '';
    }

    print('🔄 [ChatController] Iniciando envío de mensaje');
    print(
      '📄 [ChatController] Documento adjunto: ${currentPdf != null ? "Sí" : "No"}',
    );
    print('💬 [ChatController] Texto: "$cleanUserText"');

    // Update last send time for stuck detection
    lastSendTime.value = DateTime.now();

    // Validar cuota de chats antes de proceder, pero con política "permitir primero".
    // Consideramos nuevo hilo si no hay threadId o si no hay mensajes (threadId persistido podría ser viejo).
    // Consideramos nuevo hilo SOLO si no hay threadId. Tener lista de mensajes vacía ya no obliga a reiniciar.
    final isNewThread = threadId.isEmpty;
    if (!profileController.canCreateMoreChats()) {
      print(
        '⚖️ [ChatController] canCreateMoreChats=false (pre refresh), isNewThread=$isNewThread',
      );
      // Sembrar/actualizar perfil una vez para obtener cuotas actuales
      if (profileController.currentProfile.value.id <= 0 ||
          (profileController.currentProfile.value.authToken).isEmpty) {
        final basic = userService.getProfileData();
        if (basic.id > 0 && basic.authToken.isNotEmpty) {
          profileController.currentProfile.value = basic;
        }
      }
      if (profileController.currentProfile.value.id > 0 &&
          profileController.currentProfile.value.authToken.isNotEmpty) {
        await profileController.refreshProfile();
      }
      // Re-evaluar tras refrescar; si sigue sin cuota y ES hilo nuevo, informamos pero seguimos
      if (!profileController.canCreateMoreChats() && isNewThread) {
        print(
          '🚧 [ChatController] Sin cuota para crear nuevo chat; mostrando aviso pero continuando',
        );
        if (!Get.testMode)
          Get.snackbar(
            'Límite alcanzado',
            'Has alcanzado el límite de chats en tu plan actual. Actualiza tu plan para crear más chats.',
            snackPosition: SnackPosition.TOP,
            backgroundColor: Colors.orange,
            colorText: Colors.white,
            duration: const Duration(seconds: 5),
            mainButton: TextButton(
              onPressed: () => Get.toNamed(Routes.subscriptions.name),
              child: const Text(
                'Actualizar Plan',
                style: TextStyle(color: Colors.white),
              ),
            ),
          );
      }
      // Continuamos; el backend hará cumplir los límites si aplica
    }

    if (currentPdf != null && !profileController.canUploadMoreFiles()) {
      // Seed profile if it's not ready
      if (profileController.currentProfile.value.id <= 0 ||
          (profileController.currentProfile.value.authToken).isEmpty) {
        final basic = userService.getProfileData();
        if (basic.id > 0 && basic.authToken.isNotEmpty) {
          profileController.currentProfile.value = basic;
        }
      }
      if (profileController.currentProfile.value.id > 0 &&
          profileController.currentProfile.value.authToken.isNotEmpty) {
        await profileController.refreshProfile();
      }
      // Si sigue sin cuota, bloquear solo si no es un hilo nuevo; si es nuevo, permitir para evitar falsos negativos
      if (!profileController.canUploadMoreFiles() && !isNewThread) {
        if (!Get.testMode)
          Get.snackbar(
            'Límite alcanzado',
            'Has alcanzado el límite de archivos PDF en tu plan actual. Actualiza tu plan para subir más archivos.',
            snackPosition: SnackPosition.TOP,
            backgroundColor: Colors.orange,
            colorText: Colors.white,
            duration: const Duration(seconds: 5),
            mainButton: TextButton(
              onPressed: () => Get.toNamed(Routes.subscriptions.name),
              child: const Text(
                'Actualizar Plan',
                style: TextStyle(color: Colors.white),
              ),
            ),
          );
        return;
      }
    }

    try {
      isSending.value = true;
      isTyping.value = true;
      print(
        '✅ [ChatController] Estados inicializados: isSending=${isSending.value}, isTyping=${isTyping.value}',
      );

      // Si el chat es nuevo, intentar iniciar el hilo con el mensaje del usuario.
      // Si falla (p.ej., 500), reintentar con prompt vacío para obtener threadId
      // Nuevo chat únicamente cuando no hay threadId vigente.
      if (threadId.isEmpty) {
        Logger.debug('new chat start');
        try {
          _startingNewChat = true;
          Logger.debug('startChat(effectiveText)');
          final start = await chatsService.startChat(effectiveText);
          threadId = start.threadId;
          threadPref.setValue(threadId);

          currentChat.value = await chatsService.generateNewChat(
            currentChat.value,
            effectiveText,
            currentPdf, // incluir archivo para shortTitle si aplica
            threadId,
          );

          // Chat quota consumed server-side (chat_message flow)
          // Validar PDF si existe antes de enviar primer mensaje
          if (currentPdf != null) {
            Logger.debug('validating PDF (new chat)');
            isUploadingPdf.value = true;
            await attachmentService.validateFile(currentPdf);
            isUploadingPdf.value = false;
          }

          final userMessage = ChatMessageModel.user(
            chatId: currentChat.value.uid,
            text: displayText, // Mostrar texto simple en UI
            attach: currentPdf,
            imageAttach: currentImage,
          );
          messages.add(userMessage);

          // 💾 Guardar mensaje del usuario en BD local
          print('💾 [ChatController] Guardando mensaje del usuario en BD');
          await chatsService.chatMessagesLocalData.insertOne(userMessage);

          // Si hay PDF o el backend no envía texto inicial, procedemos a enviar mensaje (streaming) inmediatamente.
          final aiMessage = ChatMessageModel.ai(
            chatId: currentChat.value.uid,
            text: '',
          );
          var hasFirstToken = false;

          final response = await chatsService.sendMessage(
            threadId: threadId,
            userMessage: ChatMessageModel.user(
              chatId: userMessage.chatId,
              text: effectiveText, // Enviar texto efectivo al backend
              attach: userMessage.attach,
              imageAttach: userMessage.imageAttach,
            ),
            file: currentPdf,
            image: currentImage,
            focusDocId: focusOnPdfMode.value ? focusedPdfId : null,
            onStream: (token) {
              _processStreamToken(
                token,
                aiMessage,
                () => hasFirstToken,
                (val) => hasFirstToken = val,
              );
            },
          );
          Logger.debug('stream complete len=${response.text.length}');

          if (!hasFirstToken) {
            aiMessage.text = response.text;
            messages.add(aiMessage);
            Logger.debug('no tokens streamed; set full text');
          } else {
            // Don't overwrite the streamed content with response.text!
            // aiMessage.text = response.text;

            // DEBUG CRÍTICO: Verificar texto final acumulado
            final totalNewlines = '\n'.allMatches(aiMessage.text).length;
            final preview =
                aiMessage.text.length > 500
                    ? aiMessage.text.substring(0, 500).replaceAll('\n', '\\n')
                    : aiMessage.text.replaceAll('\n', '\\n');
            print(
              '🎯 [Controller] Stream complete. Final text length: ${aiMessage.text.length} | Newlines: $totalNewlines',
            );
            print('🎯 [Controller] Preview (first 500 chars): "$preview"');
          }

          // Debug: Verificar que no estamos mutando mensajes anteriores
          print('🔍 [Controller] Total messages in list: ${messages.length}');
          if (messages.length >= 2) {
            print(
              '🔍 [Controller] Last AI message length: ${aiMessage.text.length}',
            );
            print(
              '🔍 [Controller] Previous message (index ${messages.length - 2}) length: ${messages[messages.length - 2].text.length}',
            );
          }

          // Guardar mensaje en BD
          chatsService.chatMessagesLocalData.insertOne(aiMessage);
          pendingPdf.value = null;
          pendingImage.value = null;
          // Descontar cuota de archivo si se usó
          // File quota consumed on backend
          return;
        } catch (e) {
          // Fallback: crear thread con prompt vacío y luego enviar el mensaje del usuario vía streaming
          Logger.warn('fallback empty start after error: $e');
          // Si ya teníamos un threadId (el Start inicial sí funcionó) entonces NO debemos volver a llamar a /conversations/start
          // La excepción casi siempre proviene del primer sendMessage (p.ej. 403 de quota u otro error de red)
          // Reintentar start provoca duplicación de hilos y doble tarjeta de PDF.
          if (threadId.isNotEmpty) {
            _startingNewChat = false;
            Logger.warn('abort duplicate start: existing threadId=$threadId');
            messages.add(
              ChatMessageModel.temporal(
                'No se pudo enviar el mensaje (posible límite de cuota o error de red). Intenta de nuevo.',
                true,
              ),
            );
            scrollToBottom();
            return; // salimos sin crear un nuevo thread duplicado
          }
          if (_startingNewChat) {
            Logger.warn('avoid duplicate start (flag still set)');
            _startingNewChat = false;
          }
          final start = await chatsService.startChat('');
          threadId = start.threadId;
          threadPref.setValue(threadId);

          currentChat.value = await chatsService.generateNewChat(
            currentChat.value,
            cleanUserText,
            null,
            threadId,
          );

          // Validar PDF si existe
          if (currentPdf != null) {
            Logger.debug('validating PDF (fallback)');
            isUploadingPdf.value = true;
            await attachmentService.validateFile(currentPdf);
            isUploadingPdf.value = false;
          }

          final userMessage = ChatMessageModel.user(
            chatId: currentChat.value.uid,
            text: displayText, // Mostrar texto simple en UI
            attach: currentPdf,
            imageAttach: currentImage,
          );

          // Reset scroll state to allow auto scrolling
          resetAutoScroll();

          messages.add(userMessage);

          // 💾 Guardar mensaje del usuario en BD local (fallback path)
          print(
            '💾 [ChatController] Guardando mensaje del usuario en BD (fallback)',
          );
          await chatsService.chatMessagesLocalData.insertOne(userMessage);

          pendingPdf.value = null;
          pendingImage.value = null;
          scrollToBottom();

          // Mostrar burbuja de AI y transmitir tokens
          final aiMessage = ChatMessageModel.ai(
            chatId: currentChat.value.uid,
            text: '',
          );
          var hasFirstToken = false;

          final response = await chatsService.sendMessage(
            threadId: threadId,
            userMessage: ChatMessageModel.user(
              chatId: userMessage.chatId,
              text: effectiveText, // Enviar texto efectivo al backend
              attach: userMessage.attach,
              imageAttach: userMessage.imageAttach,
            ),
            file: currentPdf,
            image: currentImage,
            onStream: (token) {
              _processStreamToken(
                token,
                aiMessage,
                () => hasFirstToken,
                (val) => hasFirstToken = val,
              );
            },
          );
          if (!hasFirstToken) {
            aiMessage.text = response.text;
            messages.add(aiMessage);
          } else {
            // Don't overwrite the streamed content with response.text!
            // aiMessage.text = response.text;
          }

          // Quotas consumed exclusively on backend now

          return;
        }
      }
      // Resetea flag de inicio si se completó este bloque
      _startingNewChat = false;

      // Solo generar registro de chat si aún no existe uno persistido (currentChat.shortTitle vacío indica nuevo)
      if (currentChat.value.shortTitle.isEmpty) {
        Logger.debug('generate new chat with doc');
        currentChat.value = await chatsService.generateNewChat(
          currentChat.value,
          cleanUserText,
          currentPdf,
          threadId,
        );

        // Chat quota consumed server-side
      }

      if (currentPdf != null) {
        Logger.debug('validating PDF');
        isUploadingPdf.value = true;
        await attachmentService.validateFile(currentPdf);
        isUploadingPdf.value = false;
        Logger.debug('PDF validated');
      }

      final userMessage = ChatMessageModel.user(
        chatId: currentChat.value.uid,
        text: displayText, // Usar displayText en lugar de effectiveText
        attach: currentPdf,
        imageAttach: currentImage,
      );

      // Reset scroll state to allow auto scrolling
      resetAutoScroll();

      messages.add(userMessage);

      // 💾 Guardar mensaje del usuario en BD local (existing chat path)
      print(
        '💾 [ChatController] Guardando mensaje del usuario en BD (existing chat)',
      );
      await chatsService.chatMessagesLocalData.insertOne(userMessage);

      pendingPdf.value = null;
      pendingImage.value = null;
      scrollToBottom();
      Logger.debug('user message added');

      // Mantener isTyping.value = true durante el envío del documento
      // para que se muestre el indicador de escritura
      if (currentPdf != null) {
        // Asegurar que el indicador de escritura se mantenga visible
        isTyping.value = true;
        Logger.debug('keep typing indicator (PDF)');
      }

      try {
        Logger.debug('sending message (with retries)');
        final aiMessage = ChatMessageModel.ai(
          chatId: currentChat.value.uid,
          text: '',
        );
        var hasFirstToken = false;

        Future<ChatMessageModel> attemptSend() => chatsService.sendMessage(
          threadId: threadId,
          userMessage: ChatMessageModel.user(
            chatId: userMessage.chatId,
            text:
                effectiveText, // Enviar prompt vacío para PDFs, el backend lo manejará
            attach: userMessage.attach,
            imageAttach: userMessage.imageAttach,
          ),
          file: currentPdf,
          image: currentImage,
          focusDocId: focusOnPdfMode.value ? focusedPdfId : null,
          onStream: (token) {
            lastSendTime.value = DateTime.now();
            _processStreamToken(
              token,
              aiMessage,
              () => hasFirstToken,
              (val) => hasFirstToken = val,
            );
          },
        );
        ChatMessageModel response;
        const maxPdfPollRetries = 4;
        int tries = 0;
        while (true) {
          tries++;
          try {
            response = await Future.any([
              attemptSend(),
              Future.delayed(
                const Duration(minutes: 3),
              ).then((_) => throw Exception('Request timeout after 3 minutes')),
            ]);
            break; // éxito
          } catch (err) {
            final msg = err.toString();
            if (msg.contains('PDF_PROCESSING') && tries <= maxPdfPollRetries) {
              messages.add(
                ChatMessageModel.temporal(
                  'Procesando el PDF... reintentando (#$tries)',
                  true,
                ),
              );
              scrollToBottom();
              await Future.delayed(Duration(seconds: 3 * tries));
              continue;
            }
            if (msg.contains('QUOTA_EXCEEDED')) {
              messages.add(
                ChatMessageModel.temporal(
                  'Límite alcanzado para este tipo de mensaje. Actualiza tu plan.',
                  true,
                ),
              );
              scrollToBottom();
              return;
            }
            rethrow;
          }
        }
        if (!hasFirstToken) {
          aiMessage.text = response.text;
          messages.add(aiMessage);
        } else {
          // Don't overwrite the streamed content with response.text!
          // aiMessage.text = response.text;
        }
        Logger.debug('server response ok');

        // Guardar mensaje en BD
        chatsService.chatMessagesLocalData.insertOne(aiMessage);
        pendingPdf.value = null;
        pendingImage.value = null;

        // Debug: Verificar estado de mensajes después de persistencia
        print(
          '🔍 [Controller-R2] After persist - Total messages: ${messages.length}',
        );
        print(
          '🔍 [Controller-R2] After persist - AI message length: ${aiMessage.text.length}',
        );

        // Ensure proper scrolling for structured content with multiple delayed attempts
        scrollToBottom();
        // Schedule additional scrolls for complex markdown content
        Future.delayed(const Duration(milliseconds: 800), scrollToBottom);
        Future.delayed(const Duration(seconds: 1), scrollToBottom);

        // File quota consumed server-side
      } catch (error) {
        Logger.error('sendMessage error: $error');
        _lastFailedUserMessage = userMessage;
        _lastFailedPdf = currentPdf;
        messages.add(
          ChatMessageModel.temporal(
            '¡Ups! Parece que hubo un problema al procesar tu mensaje. Por favor, intenta nuevamente en unos momentos.',
            true,
          ),
        );
        scrollToBottom();
      }
    } catch (e) {
      Logger.error('general error: $e threadId=$threadId');
      debugPrint('Error sending message: $e');
    } finally {
      isSending.value = false;
      isTyping.value = false;
      Logger.debug(
        'final states isSending=${isSending.value} isTyping=${isTyping.value}',
      );
    }
  }

  void scrollToBottom() {
    if (_chatScrollController == null || !_chatScrollController!.hasClients) {
      return;
    }

    // First immediate scroll to current max extent
    WidgetsBinding.instance.addPostFrameCallback((_) {
      try {
        final position = _chatScrollController!.position;

        // Skip if user manually scrolled and we're not forcing scroll
        if (userManuallyScrolled.value && !autoScrollEnabled.value) return;

        final target = position.maxScrollExtent;
        final duration =
            position.maxScrollExtent - position.pixels > 1000
                ? const Duration(milliseconds: 500)
                : const Duration(milliseconds: 200);

        _chatScrollController?.animateTo(
          target,
          duration: duration,
          curve: Curves.easeOutCubic,
        );

        // Add a second delayed scroll to handle dynamically rendered content
        Future.delayed(const Duration(milliseconds: 500), () {
          if (_chatScrollController == null ||
              !_chatScrollController!.hasClients)
            return;

          try {
            // Recheck maxScrollExtent after layout is complete
            final updatedPosition = _chatScrollController!.position;

            // Only scroll if we're not already at the bottom and auto-scroll is enabled
            if (updatedPosition.pixels < updatedPosition.maxScrollExtent &&
                (autoScrollEnabled.value || !userManuallyScrolled.value)) {
              _chatScrollController?.animateTo(
                updatedPosition.maxScrollExtent,
                duration: const Duration(milliseconds: 300),
                curve: Curves.easeOutCubic,
              );
            }
          } catch (e) {
            // Ignore scroll errors
          }
        });
      } catch (e) {
        // Ignore scroll errors
      }
    });
  }

  void resetAutoScroll() {
    // Re-enable auto scrolling (typically after user sends a message)
    autoScrollEnabled.value = true;
    userManuallyScrolled.value = false;
  }

  void setFocusNode(FocusNode? focusNode) {
    _focusNode = focusNode;
  }

  void setScrollController(ScrollController? scrollController) {
    _chatScrollController = scrollController;
  }

  void focusOnChatInputText() {
    if (_focusNode != null) {
      try {
        _focusNode!.requestFocus();
      } catch (e) {
        Logger.error(e.toString());
      }
    }
  }

  void showChat(String chatId) async {
    try {
      print('🔍 [ChatController] showChat llamado con chatId: $chatId');

      loading.value = true;
      messages.clear();

      // Cerrar drawer/overlays
      Get.back(closeOverlays: true);

      // Navegar a la vista del chat si no estamos ahí
      if (Get.currentRoute != Routes.home.name) {
        await Get.toNamed(Routes.home.name);
      }

      // Cargar chat y mensajes de forma secuencial con manejo de errores
      try {
        await _loadChatById(chatId);
        print('✅ [ChatController] Chat cargado: ${currentChat.value.uid}');

        await _loadMessagesByChatId(chatId);
        print('✅ [ChatController] Mensajes cargados: ${messages.length}');

        // Scroll al final después de cargar
        WidgetsBinding.instance.addPostFrameCallback((_) {
          scrollToBottom();
        });
      } catch (e) {
        print('❌ [ChatController] Error al cargar chat/mensajes: $e');
        error.value = 'Error al cargar el chat';
        Notify.snackbar(
          'Error',
          'No se pudo cargar el chat. Inténtalo de nuevo.',
          NotifyType.error,
        );
      }

      loading.value = false;
    } catch (e) {
      print('❌ [ChatController] Error general en showChat: $e');
      Notify.snackbar('Error', 'No se encontró el chat.', NotifyType.error);
      loading.value = false;
    }
  }
}
