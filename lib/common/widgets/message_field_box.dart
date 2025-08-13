import 'package:ema_educacion_medica_avanzada/app/chat/controllers/chat_controller.dart';
import 'package:ema_educacion_medica_avanzada/common/widgets/app_icons.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:ema_educacion_medica_avanzada/core/core.dart';
import 'package:flutter/material.dart';
import 'package:get/get_state_manager/src/rx_flutter/rx_obx_widget.dart';

class MessageFieldBox extends StatefulWidget {
  final ChatController chatController;
  final NavigationService navigatioService;
  final String title;

  MessageFieldBox({
    super.key,
    required this.chatController,
    required this.navigatioService,
    this.title = 'Pregúntame lo que quieras...',
  });

  @override
  State<MessageFieldBox> createState() => _MessageFieldBoxState();
}

class _MessageFieldBoxState extends State<MessageFieldBox> {
  late final TextEditingController _textController;
  late final FocusNode _focusNode;

  final _outlineEnableBorder = OutlineInputBorder(
    borderRadius: BorderRadius.circular(21),
    borderSide: BorderSide(color: Colors.transparent),
  );
  final _outlineFocusBorder = OutlineInputBorder(
    borderRadius: BorderRadius.circular(21),
    borderSide: BorderSide(color: AppStyles.primary900),
  );

  @override
  void initState() {
    super.initState();
    _textController = TextEditingController();
    _focusNode = FocusNode();
    widget.chatController.setFocusNode(_focusNode);
  }

  @override
  void dispose() {
    _textController.dispose();
    _focusNode.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Obx(() {
      final sending = widget.chatController.isSending.value;

      final buttons = Padding(
        padding: const EdgeInsets.only(right: 6),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            IconButton(
              onPressed:
                  sending
                      ? null
                      : () {
                        widget.navigatioService.goTo(
                          OverlayRoutes.pdfUpdloader,
                        );
                      },
              icon: AppIcons.attachFile(
                color: AppStyles.tertiaryColor,
                height: 30,
                width: 30,
              ),
            ),
            IconButton.filled(
              onPressed:
                  sending
                      ? null
                      : () {
                        widget.chatController.sendMessage(
                          _textController.value.text,
                        );
                        _textController.clear();
                        _focusNode.requestFocus();
                      },
              padding: const EdgeInsets.all(8),
              style: ButtonStyle(
                backgroundColor: const WidgetStatePropertyAll(
                  AppStyles.tertiaryColor,
                ),
              ),
              icon:
                  sending
                      ? const SizedBox(
                        width: 18,
                        height: 18,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          valueColor: AlwaysStoppedAnimation<Color>(
                            AppStyles.whiteColor,
                          ),
                        ),
                      )
                      : AppIcons.cornerDownLeft(
                        height: 18,
                        width: 18,
                        color: AppStyles.whiteColor,
                      ),
            ),
          ],
        ),
      );

      final inputDecoration = InputDecoration(
        label: Text(widget.title),
        enabledBorder: _outlineEnableBorder,
        focusedBorder: _outlineFocusBorder,
        floatingLabelBehavior: FloatingLabelBehavior.never,
        suffixIcon: buttons,
        filled: true,
      );

      return TextFormField(
        autocorrect: false,
        enableSuggestions: true,
        textCapitalization: TextCapitalization.sentences,
        focusNode: _focusNode,
        controller: _textController,
        style: AppStyles.chatMessageUser,
        decoration: inputDecoration,
        enabled: !sending,
        keyboardType: TextInputType.text,
        maxLines: null,
        onFieldSubmitted:
            sending
                ? null
                : (value) {
                  widget.chatController.sendMessage(value);
                  _textController.clear();
                  _focusNode.requestFocus();
                },
      );
    });
  }
}
