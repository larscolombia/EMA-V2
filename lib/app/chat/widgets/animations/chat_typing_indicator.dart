import 'package:flutter/material.dart';


class ChatTypingIndicator extends StatefulWidget {
  const ChatTypingIndicator({super.key});

  @override
  State<ChatTypingIndicator> createState() => _ChatTypingIndicatorState();
}

class _ChatTypingIndicatorState extends State<ChatTypingIndicator>
    with SingleTickerProviderStateMixin {
  late AnimationController _controller;

  @override
  void initState() {
    super.initState();
    print('📝 [ChatTypingIndicator] Inicializando indicador de escritura');
    _controller = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 1000),
    )..repeat();
  }

  @override
  void dispose() {
    print('📝 [ChatTypingIndicator] Disposing indicador de escritura');
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    print('📝 [ChatTypingIndicator] Renderizando indicador de escritura');
    return Align(
      alignment: Alignment.centerLeft,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 12),
        margin: const EdgeInsets.only(right: 24, left: 12),
        decoration: BoxDecoration(
          color: const Color.fromRGBO(58, 12, 140, 0.9),
          borderRadius: BorderRadius.circular(12),
        ),
        child: IntrinsicWidth(
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              _DotWidget(
                  controller: _controller, interval: const Interval(0.0, 0.3)),
              const SizedBox(width: 3),
              _DotWidget(
                  controller: _controller, interval: const Interval(0.3, 0.6)),
              const SizedBox(width: 3),
              _DotWidget(
                  controller: _controller, interval: const Interval(0.6, 0.9)),
            ],
          ),
        ),
      ),
    );
  }
}

class _DotWidget extends StatelessWidget {
  final AnimationController controller;
  final Interval interval;

  const _DotWidget({
    required this.controller,
    required this.interval,
  });

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: controller,
      builder: (context, child) {
        final animation = CurvedAnimation(parent: controller, curve: interval);
        return Transform.translate(
          offset: Offset(0, -2 * animation.value),
          child: Container(
            width: 6,
            height: 6,
            decoration: const BoxDecoration(
              color: Colors.white,
              shape: BoxShape.circle,
            ),
          ),
        );
      },
    );
  }
}
