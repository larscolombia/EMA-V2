import 'package:ema_educacion_medica_avanzada/app/chat/chat.dart';
import 'package:ema_educacion_medica_avanzada/app/chat/controllers/chat_controller.dart';
import 'package:ema_educacion_medica_avanzada/common/screens.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:ema_educacion_medica_avanzada/core/core.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';

import '../models/chat_message_model.dart';
import '../widgets/animations/chat_typing_indicator.dart';
import '../widgets/animations/pending_pdf_preview.dart';
import '../widgets/chat_error_message.dart';

class ChatHomeView extends StatefulWidget {
  const ChatHomeView({super.key});

  @override
  State<ChatHomeView> createState() => _ChatHomeViewState();
}

class _ChatHomeViewState extends State<ChatHomeView>
    with TickerProviderStateMixin {
  final GlobalKey<ScaffoldState> _scaffoldKey = GlobalKey<ScaffoldState>();
  final chatController = Get.find<ChatController>();
  final navigationService = Get.find<NavigationService>();
  final scrollController = ScrollController();
  bool isUserScrolling = false;

  @override
  void initState() {
    super.initState();
    chatController.setScrollController(scrollController);

    // The chat will start once the user sends the first message

    // Setup scroll listener
    scrollController.addListener(_scrollListener);

    // Initial scroll to bottom
    WidgetsBinding.instance.addPostFrameCallback((_) {
      chatController.scrollToBottom();
    });
  }

  @override
  void dispose() {
    scrollController.removeListener(_scrollListener);
    chatController.setScrollController(null);
    super.dispose();
  }

  void _scrollListener() {
    // Check if we're at the bottom
    if (scrollController.hasClients) {
      final maxScroll = scrollController.position.maxScrollExtent;
      final currentScroll = scrollController.position.pixels;

      // We're near the bottom, enable auto-scroll
      if (maxScroll - currentScroll <= 100) {
        chatController.autoScrollEnabled.value = true;
        chatController.userManuallyScrolled.value = false;
      }
    }
  }

  Widget _buildMessageWidget(ChatMessageModel message) {
    if (message.widget != null) {
      return message.widget!;
    }

    // Mejorado el manejo de mensajes cancelados y errores
    if (message.aiMessage) {
      if (message.text.contains('cancelado')) {
        return ChatErrorMessage(
          message: 'Mensaje cancelado',
          onRetry: () => chatController.cleanChat(),
        );
      } else if (message.text.contains('Error') ||
          message.text.contains('¡Ups!')) {
        return ChatErrorMessage(
          message: message.text,
          onRetry: () => chatController.retryLastSend(),
        );
      }
    }

    return message.aiMessage
        ? ChatMessageAi(key: ValueKey(message.uid), message: message)
        : ChatMessageUser(key: ValueKey(message.uid), message: message);
  }

  Widget _buildChatList() {
    return Obx(() {
      final msgs = chatController.messages;
      // Cambiamos AnimatedList por ListView para un mejor control del scroll
      return ListView.builder(
        controller: scrollController,
        itemCount: msgs.length + (chatController.isTyping.value ? 1 : 0),
        padding: const EdgeInsets.only(left: 12, right: 12, top: 16),
        itemBuilder: (context, index) {
          if (index < msgs.length) {
            final message = msgs[index];
            return AnimatedSize(
              key: ValueKey('animated_${message.uid}'), // Key única para AnimatedSize
              duration: const Duration(milliseconds: 300),
              curve: Curves.easeOutCubic,
              child: AnimatedOpacity(
                duration: const Duration(milliseconds: 300),
                opacity: 1.0,
                child: _buildMessageWidget(message),
              ),
            );
          } else {
            // Show typing indicator with timeout detection
            return Obx(() {
              final timeSinceLastSend = DateTime.now().difference(
                chatController.lastSendTime.value,
              );
              final isLongRunning = timeSinceLastSend.inSeconds > 30;

              return Padding(
                padding: EdgeInsets.only(top: 8.0),
                child: Column(
                  children: [
                    Obx(() {
                      final stage = chatController.currentStage.value;
                      // Map backend stages to user-friendly captions
                      final captions =
                          <String>[
                            'Procesando…',
                            if (stage == 'rag_search' || stage.isEmpty)
                              'Analizando vector…',
                            if (stage == 'doc_only')
                              'Usando documentos adjuntos…',
                            if (stage == 'pubmed_search') 'Buscando en PubMed…',
                            if (stage == 'rag_found')
                              'Fuente interna encontrada…',
                            if (stage == 'pubmed_found')
                              'Referencia PubMed encontrada…',
                            'Enviando respuesta…',
                          ].where((e) => e.isNotEmpty).toList();
                      return ChatTypingIndicator(
                        captions: captions,
                        captionInterval: const Duration(seconds: 2),
                      );
                    }),
                    if (isLongRunning) ...[
                      const SizedBox(height: 8),
                      Container(
                        padding: const EdgeInsets.all(12),
                        margin: const EdgeInsets.symmetric(horizontal: 16),
                        decoration: BoxDecoration(
                          color: Colors.orange.shade50,
                          borderRadius: BorderRadius.circular(8),
                          border: Border.all(color: Colors.orange.shade200),
                        ),
                        child: Row(
                          children: [
                            Icon(
                              Icons.warning_amber,
                              color: Colors.orange.shade600,
                              size: 20,
                            ),
                            const SizedBox(width: 8),
                            Expanded(
                              child: Text(
                                'La respuesta está tomando más tiempo del esperado...',
                                style: TextStyle(
                                  color: Colors.orange.shade700,
                                  fontSize: 12,
                                ),
                              ),
                            ),
                            TextButton(
                              onPressed:
                                  () => chatController.forceStopAndReset(),
                              style: TextButton.styleFrom(
                                padding: const EdgeInsets.symmetric(
                                  horizontal: 8,
                                ),
                                minimumSize: const Size(0, 32),
                              ),
                              child: Text(
                                'Detener',
                                style: TextStyle(
                                  color: Colors.orange.shade700,
                                  fontSize: 12,
                                  fontWeight: FontWeight.w600,
                                ),
                              ),
                            ),
                          ],
                        ),
                      ),
                    ],
                  ],
                ),
              );
            });
          }
        },
      );
    });
  }

  @override
  Widget build(BuildContext context) {
    chatController.setScrollController(scrollController);

    final actions = Column(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        Text('Acciones rápidas', style: AppStyles.homeTitle),
        const SizedBox(height: 32),
        Row(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            OutlinedButton.icon(
              onPressed: () {
                navigationService.goTo(OverlayRoutes.homeQuizzesMenu);
              },
              style: ButtonStyle(
                side: WidgetStateProperty.all(
                  BorderSide(color: AppStyles.tertiaryColor),
                ),
              ),
              icon: AppIcons.userSquare(height: 18, width: 18),
              label: const Text('Cuestionarios'),
            ),
            const SizedBox(width: 10),
            OutlinedButton.icon(
              onPressed: () {
                navigationService.goTo(OverlayRoutes.homeClinicalCasesMenu);
              },
              style: ButtonStyle(
                side: WidgetStateProperty.all(
                  BorderSide(color: AppStyles.tertiaryColor),
                ),
              ),
              icon: AppIcons.bussinessBag(
                color: AppStyles.secondaryColor,
                height: 18,
                width: 18,
              ),
              label: const Text('Casos Clínicos'),
            ),
          ],
        ),
      ],
    );

    return AppLayout(
      key: _scaffoldKey,
      body: Column(
        children: [
          Expanded(
            child: NotificationListener<ScrollNotification>(
              onNotification: (notification) {
                if (notification is ScrollUpdateNotification) {
                  if (notification.dragDetails != null) {
                    // User is manually scrolling, disable auto-scroll
                    chatController.userManuallyScrolled.value = true;

                    // If the user scrolls up, disable auto-scroll
                    if (notification.dragDetails!.primaryDelta! > 0) {
                      chatController.autoScrollEnabled.value = false;
                    }
                  }
                }
                return true;
              },
              child: Obx(() {
                final showChat = chatController.messages.isNotEmpty;
                return showChat ? _buildChatList() : actions;
              }),
            ),
          ),
          Obx(() {
            final pendingPdf = chatController.pendingPdf.value;
            return Column(
              children: [
                if (pendingPdf != null)
                  PendingPdfPreview(
                    pdf: pendingPdf,
                    onRemove: () => chatController.pendingPdf.value = null,
                    isUploading: chatController.isUploadingPdf.value,
                  ),
                AppScaffold.bottomFieldBox(chatController, navigationService),
              ],
            );
          }),
        ],
      ),
    );
  }
}
